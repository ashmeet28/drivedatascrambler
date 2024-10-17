package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strconv"
)

func createChuckPairs(chuckSize uint64, driveSize uint64) []uint64 {
	var chuckPairs []uint64
	for i := uint64(0); i < (driveSize / chuckSize); i++ {
		chuckPairs = append(chuckPairs, i)
	}
	return chuckPairs
}

func getRandomIntToShuffleChuckPairs(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		fmt.Println("Error")
		os.Exit(1)
	}
	if !n.IsInt64() {
		fmt.Println("Error")
		os.Exit(1)
	}
	return int(n.Int64())
}

func shuffleChuckPairs(chuckPairs []uint64) []uint64 {
	l := len(chuckPairs)

	for i := 0; i < l; i++ {
		j := getRandomIntToShuffleChuckPairs(l)
		t := chuckPairs[j]
		chuckPairs[j] = chuckPairs[i]
		chuckPairs[i] = t
	}

	return chuckPairs
}

func putFirstAndLastChuckPairsInFront(chuckPairs []uint64) []uint64 {
	l := len(chuckPairs)

	for i := 0; i < l; i++ {
		if chuckPairs[i] == 0 {
			t := chuckPairs[0]
			chuckPairs[0] = chuckPairs[i]
			chuckPairs[i] = t
			break
		}
	}
	for i := 0; i < l; i++ {
		if chuckPairs[i] == uint64(l-1) {
			t := chuckPairs[2]
			chuckPairs[2] = chuckPairs[i]
			chuckPairs[i] = t
			break
		}
	}

	return chuckPairs
}

func createXORKey() []byte {
	bytes := make([]byte, 16)

	if _, err := rand.Read(bytes); err != nil {
		fmt.Println("Error")
		os.Exit(1)
	}

	return bytes
}

func createBashCommand(drivePath string,
	firstChuckFilePath string, secondChuckFilePath string,
	xorFirstChuckFilePath string, xorSecondChuckFilePath string,
	chuckSize uint64, firstChuck int, secondChuck int) []byte {

	var buf []byte
	var lineBytes []byte

	lineBytes = []byte("sudo dd if=" + drivePath + " of=" + firstChuckFilePath +
		" bs=" + strconv.FormatUint(chuckSize, 10) +
		" skip=" + strconv.FormatUint(uint64(firstChuck), 10) + " count=1")

	buf = append(buf, lineBytes...)
	buf = append(buf, 0x0a)

	lineBytes = []byte("sudo dd if=" + drivePath + " of=" + secondChuckFilePath +
		" bs=" + strconv.FormatUint(chuckSize, 10) +
		" skip=" + strconv.FormatUint(uint64(secondChuck), 10) + " count=1")

	buf = append(buf, lineBytes...)
	buf = append(buf, 0x0a)

	lineBytes = []byte("drivedatascrambler xorwithkey " +
		hex.EncodeToString(createXORKey()) + " " +
		firstChuckFilePath + " " + secondChuckFilePath + " " +
		xorFirstChuckFilePath + " " + xorSecondChuckFilePath)

	buf = append(buf, lineBytes...)
	buf = append(buf, 0x0a)

	lineBytes = []byte("sudo rm " + firstChuckFilePath + " " + secondChuckFilePath)
	buf = append(buf, lineBytes...)
	buf = append(buf, 0x0a)

	lineBytes = []byte("sudo dd if=" + xorFirstChuckFilePath + " of=" + drivePath +
		" bs=" + strconv.FormatUint(chuckSize, 10) +
		" seek=" + strconv.FormatUint(uint64(firstChuck), 10) + " count=1" + " conv=notrunc")

	buf = append(buf, lineBytes...)
	buf = append(buf, 0x0a)

	lineBytes = []byte("sudo dd if=" + xorSecondChuckFilePath + " of=" + drivePath +
		" bs=" + strconv.FormatUint(chuckSize, 10) +
		" seek=" + strconv.FormatUint(uint64(secondChuck), 10) + " count=1" + " conv=notrunc")

	buf = append(buf, lineBytes...)
	buf = append(buf, 0x0a)

	lineBytes = []byte("rm " + xorFirstChuckFilePath + " " + xorSecondChuckFilePath)
	buf = append(buf, lineBytes...)
	buf = append(buf, 0x0a)

	lineBytes = []byte("sudo sync")
	buf = append(buf, lineBytes...)
	buf = append(buf, 0x0a)

	return buf
}

func xorChuckWithKey(k string, c []byte) []byte {
	keyBytes, err := hex.DecodeString(k)
	if err != nil {
		fmt.Println("Invalid Command")
		os.Exit(1)
	}

	for i := range c {
		c[i] = c[i] ^ keyBytes[i%len(keyBytes)]
	}

	return c
}

func main() {
	invocationCommand := os.Args[1]

	switch invocationCommand {
	case "genbashfiles":
		chuckSize, err := strconv.ParseUint(os.Args[2], 10, 64)
		if err != nil {
			fmt.Println("Invalid Command")
			os.Exit(1)
		}

		driveSize, err := strconv.ParseUint(os.Args[3], 10, 64)
		if err != nil {
			fmt.Println("Invalid Command")
			os.Exit(1)
		}

		chuckPairs := putFirstAndLastChuckPairsInFront(shuffleChuckPairs(
			createChuckPairs(chuckSize, driveSize)))

		drivePath := os.Args[4]
		firstChuckFilePath := os.Args[5]
		secondChuckFilePath := os.Args[6]
		xorFirstChuckFilePath := os.Args[7]
		xorSecondChuckFilePath := os.Args[8]

		initBashFilePath := os.Args[9]
		continueBashFilePath := os.Args[10]

		var buf []byte

		for i := 0; i < len(chuckPairs); i += 2 {
			buf = append(buf, createBashCommand(drivePath,
				firstChuckFilePath, secondChuckFilePath,
				xorFirstChuckFilePath, xorSecondChuckFilePath,
				chuckSize,
				int(chuckPairs[i]), int(chuckPairs[i+1]))...)

			if i == 2 {
				os.WriteFile(initBashFilePath, buf, 0644)
				buf = make([]byte, 0)
			}
		}

		os.WriteFile(continueBashFilePath, buf, 0644)

	case "xorwithkey":
		xorKey := os.Args[2]

		firstChuckFilePath := os.Args[3]
		secondChuckFilePath := os.Args[4]
		xorFirstChuckFilePath := os.Args[5]
		xorSecondChuckFilePath := os.Args[6]

		b1, err := os.ReadFile(firstChuckFilePath)
		if err != nil {
			fmt.Println("Error")
			os.Exit(1)
		}

		os.WriteFile(xorFirstChuckFilePath, xorChuckWithKey(xorKey, b1), 0644)

		b2, err := os.ReadFile(secondChuckFilePath)
		if err != nil {
			fmt.Println("Error")
			os.Exit(1)
		}

		os.WriteFile(xorSecondChuckFilePath, xorChuckWithKey(xorKey, b2), 0644)

	case "help":

	default:
		fmt.Println("Invalid Command")
	}
	// firstFilePath := os.Args[1]
	// secondFilePath := os.Args[2]
}
