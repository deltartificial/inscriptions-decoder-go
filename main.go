package main

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	pointer   int
	inputData string = "2024531bd7c0de19af5b3009033c130e757702f7d4b933aa848be3d825e090a041ac0063036f7264010118746578742f706c61696e3b636861727365743d7574662d38003a7b2270223a226272632d3230222c226f70223a226d696e74222c227469636b223a2273617473222c22616d74223a22313030303030303030227d68"
)

func readBytes(raw []byte, n int) []byte {
	value := raw[pointer : pointer+n]
	pointer += n
	return value
}

func getInitialPosition(raw []byte) (int, error) {
	inscriptionMark := []byte{0x00, 0x63, 0x03, 0x6f, 0x72, 0x64}
	pos := strings.Index(string(raw), string(inscriptionMark))
	if pos == -1 {
		return 0, errors.New("No ordinal inscription found in transaction")
	}
	return pos + len(inscriptionMark), nil
}

func readContentType(raw []byte) (string, error) {
	OP_1 := byte(0x51)

	b := readBytes(raw, 1)[0]
	if b != OP_1 {
		if b != 0x01 || readBytes(raw, 1)[0] != 0x01 {
			return "", errors.New("Invalid byte sequence")
		}
	}

	size := int(readBytes(raw, 1)[0])
	contentType := readBytes(raw, size)
	return string(contentType), nil
}

func readPushdata(raw []byte, opcode byte) ([]byte, error) {
	intOpcode := int(opcode)

	if 0x01 <= intOpcode && intOpcode <= 0x4b {
		return readBytes(raw, intOpcode), nil
	}

	numBytes := 0
	switch intOpcode {
	case 0x4c:
		numBytes = 1
	case 0x4d:
		numBytes = 2
	case 0x4e:
		numBytes = 4
	default:
		return nil, fmt.Errorf("Invalid push opcode %x at position %d", intOpcode, pointer)
	}

	if pointer+numBytes > len(raw) {
		return nil, fmt.Errorf("Invalid data length at position %d", pointer)
	}

	sizeBytes := readBytes(raw, numBytes)
	var size int
	switch numBytes {
	case 1:
		size = int(sizeBytes[0])
	case 2:
		size = int(binary.LittleEndian.Uint16(sizeBytes))
	case 4:
		size = int(binary.LittleEndian.Uint32(sizeBytes))
	}

	if pointer+size > len(raw) {
		return nil, fmt.Errorf("Invalid data length at position %d", pointer)
	}

	return readBytes(raw, size), nil
}

func writeDataUri(data []byte, contentType string) {
	dataBase64 := base64.StdEncoding.EncodeToString(data)
	fmt.Printf("data:%s;base64,%s", contentType, dataBase64)
}

func writeFile(data []byte, filename string) {
	if filename == "" {
		filename = "out.txt"
	}

	filename = "out/" + filename

	i := 1
	baseFilename := filename
	for _, err := os.Stat(filename); !os.IsNotExist(err); _, err = os.Stat(filename) {
		i++
		filename = fmt.Sprintf("%s%d", baseFilename, i)
	}

	fmt.Printf("Writing contents to file \"%s\"\n", filename)
	err := os.WriteFile(filename, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	raw, err := hex.DecodeString(inputData)
	if err != nil {
		log.Fatal(err)
	}

	pointer, _ = getInitialPosition(raw)

	contentType, _ := readContentType(raw)
	fmt.Printf("Content type: %s\n", contentType)
	if readBytes(raw, 1)[0] != byte(0x00) {
		fmt.Println("Error: Invalid byte sequence")
		os.Exit(1)
	}

	data := []byte{}

	OP_ENDIF := byte(0x68)
	opcode := readBytes(raw, 1)[0]
	for opcode != OP_ENDIF {
		chunk, _ := readPushdata(raw, opcode)
		data = append(data, chunk...)
		opcode = readBytes(raw, 1)[0]
	}

	fmt.Printf("Total size: %d bytes\n", len(data))
	writeFile(data, "output")
	fmt.Println("\nDone")
}
