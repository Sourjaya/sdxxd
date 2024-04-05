package xxd

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"
)

type Flags struct {
	Endianess bool
	GroupSize string
	Length    string
	Columns   string
	Seek      string
	Revert    bool
}
type ParsedFlags struct {
	IsFile bool
	E      bool
	G      int
	L      int
	S      int
	C      int
	R      bool
}

type IsSetFlags struct {
	IsSetC bool
	IsSetL bool
	IsSetG bool
	IsSetS bool
}

func NewFlags() (*Flags, *IsSetFlags, []string) {
	flags := new(Flags)
	setFlags := &IsSetFlags{}
	flag.BoolVarP(&flags.Endianess, "little-endian", "e", false, "little-endian")
	flag.StringVarP(&flags.GroupSize, "group-size", "g", "2", "group-size")
	flag.StringVarP(&flags.Length, "length", "l", "-1", "length")
	flag.StringVarP(&flags.Columns, "cols", "c", "16", "columns")
	flag.StringVarP(&flags.Seek, "seek", "s", "0", "seek")
	flag.BoolVarP(&flags.Revert, "revert", "r", false, "revert")
	flag.Parse()
	flag.Visit(func(f *flag.Flag) {
		if f.Shorthand == "c" {
			setFlags.IsSetC = true
		}
		if f.Shorthand == "l" {
			setFlags.IsSetL = true
		}
		if f.Shorthand == "g" {
			setFlags.IsSetG = true
		}
		if f.Shorthand == "s" {
			setFlags.IsSetS = true
		}
	})
	args := flag.Args()
	return flags, setFlags, args
}

func NumberParse(input string) (res int64, err error) {
	re := regexp.MustCompile(`-?0[xX][0-9a-fA-F]+|-\b0[0-7]*\b|-\b[1-9][0-9]*\b|0[xX][0-9a-fA-F]+|\b0[0-7]*\b|\b[1-9][0-9]*\b`)
	s := re.FindString(input)
	if s != "" {
		return strconv.ParseInt(s, 0, 64)
	}
	return 0, nil
}
func inputParse(s []byte, offset int, f *ParsedFlags, length int) {
	buffer := byteToHex(s, f.C)
	dumpHex(offset, length, f, buffer, s)
}

func reverseString(input string) string {
	// Decode hex string to byte slice
	hexStr := strings.ReplaceAll(input, " ", "")
	bytes, _ := hex.DecodeString(hexStr)
	// Reverse the byte slice
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}
	// Encode the reversed byte slice back to hex string
	reversed := hex.EncodeToString(bytes)
	whitespace := strings.Repeat(" ", len(input)-len(reversed))
	return whitespace + reversed
}
func byteToHex(byteBuffer []byte, count int) string {
	encodedString := hex.EncodeToString(byteBuffer)
	for i := 0; i < (count-(len(byteBuffer)%count))*2; i++ {
		encodedString = fmt.Sprint(encodedString, " ")
	}
	return encodedString
}
func bytesToString(input []byte) string {
	output := make([]byte, len(input))
	for i, b := range input {
		if b < 0x20 || b > 0x7e {
			output[i] = '.'
		} else {
			output[i] = b
		}
	}
	return string(output)
}
func dumpHex(offset, length int, f *ParsedFlags, stringBuffer string, buffer []byte) {
	i, rowCount, groupCount := 0, 0, 0
	var groupBuffer string
	for i < length*2 {
		if !f.IsFile {
			fmt.Printf("%08x: ", (offset*f.C + f.C*rowCount + f.S))
		} else {
			fmt.Printf("%08x: ", (offset*size(f.C) + f.C*rowCount + f.S))
		}
		groupCount = 1

		for j := 0; j < f.C*2; j += f.G * 2 {
			if groupCount*f.G*2 > f.C*2 {
				groupBuffer = stringBuffer[i+j : i+(f.C*2)]
			} else {
				groupBuffer = stringBuffer[i+j : i+(groupCount*f.G*2)]
			}
			if f.E {
				groupBuffer = reverseString(groupBuffer)
			}
			fmt.Printf("%s ", groupBuffer)
			groupCount += 1

		}
		var originalBuffer string
		if (f.C * (rowCount + 1)) > len(buffer) {
			originalBuffer = bytesToString(buffer[(f.C * rowCount):])
		} else {
			originalBuffer = bytesToString(buffer[(f.C * rowCount):(f.C * (rowCount + 1))])
		}
		fmt.Printf(" %v\n", originalBuffer)
		i += f.C * 2
		rowCount += 1
	}
}

// func parseL() {}
// func parseG() {}
// func parseC() {}
// func parseS() {}

func CheckFlags(isFile bool, f *Flags, size int, setFlags *IsSetFlags) (flag *ParsedFlags) {
	flag = &ParsedFlags{}
	flag.R = f.Revert
	flag.E = f.Endianess
	flag.IsFile = isFile

	var res int64
	var err error
	if setFlags.IsSetL {
		if res, err = NumberParse(f.Length); err != nil || res == 0 {
			os.Exit(1)
		} else if res < 0 {
			flag.L = size
		} else if int(res) > size {
			if isFile {
				flag.L = size
			} else {
				flag.L = int(res)
			}
		} else {
			flag.L = int(res)
		}
	} else {
		flag.L = size
	}

	if setFlags.IsSetG && f.Endianess {
		if res, err = NumberParse(f.GroupSize); err != nil {
			flag.G = 16
		} else if res < 0 {
			flag.G = 4
		} else if res > 0 {
			if res&(res-1) == 0 {
				flag.G = int(res)
			} else {
				fmt.Println("sdxxd: number of octets per group must be a power of 2 with -e.")
				os.Exit(1)
			}
		} else {
			flag.G = 16
		}
	} else if setFlags.IsSetG {
		if res, err = NumberParse(f.GroupSize); err != nil {
			flag.G = 16
		} else if res < 0 {
			flag.G = 2
		} else if res > 0 {
			flag.G = int(res)
		} else {
			flag.G = 16
		}
	} else if f.Endianess {
		flag.G = 4
	} else {
		flag.G = 2
	}
	if setFlags.IsSetC {
		if res, err := NumberParse(f.Columns); err != nil {
			flag.C = 16
		} else {
			flag.C = int(res)
		}
	} else {
		flag.C = 16
	}
	if setFlags.IsSetS {
		if f.Seek == "-0" && !isFile {
			fmt.Fprintln(os.Stderr, "sdxxd: Sorry, cannnot seek.")
		} else if f.Seek == "-0" && isFile {
			flag.S = size
		} else if res, err := NumberParse(f.Seek); err != nil {
			flag.S = 0
		} else {
			if res < 0 {
				flag.S = size + int(res)
			} else {
				flag.S = int(res)
			}
		}
	} else {
		flag.S = 0
	}
	return flag
}
func size(cols int) int {
	i := 1
	bytes := i * cols
	if bytes > 2048 {
		return bytes
	}
	// Adjust bytesToRead within the desired range
	for bytes < 2048 {
		i += 1
		bytes = i * cols
	}
	return bytes
}
func trimWords(s string) string {
	words := strings.Fields(s)

	return strings.Join(words, "")
}

func revert(input any) error {
	var str string
	switch v := input.(type) {
	case *os.File:
		scanner := bufio.NewScanner(v)
		// Read line by line
		for {
			for scanner.Scan() {
				field := trimWords(strings.TrimSpace(strings.Split(strings.Split(scanner.Text(), ":")[1], "  ")[0]))
				str += field
				if len(str) > 4096 {
					break
				}
			}
			//fmt.Println(str)
			decodedString, err := hex.DecodeString(str)
			if err != nil {
				return errors.New("2")
			}
			os.Stdout.Write(decodedString)
			if len(str) < 4096 {
				break
			}
			str = ""
		}
	case *bufio.Scanner:
		v.Split(bufio.ScanLines)
		//var input string
		//fmt.Printf("Input is : %s", input)
		for v.Scan() {
			line := v.Text()
			//fmt.Printf("\nLine : %v", line)
			field := trimWords(strings.TrimSpace(strings.Split(strings.Split(line, ":")[1], "  ")[0]))
			//fmt.Printf("field: %v", field)
			str += field
		}
		decodedString, err := hex.DecodeString(str)
		if err != nil {
			return errors.New("2")
		}
		os.Stdout.Write(decodedString)
	}
	return nil
}
func processStdIn(f *Flags, setFlags *IsSetFlags) {
	offset := 0
	var flags *ParsedFlags
	if f.Revert {
		//fmt.Println(s)
		scanner := bufio.NewScanner(os.Stdin)
		revert(scanner)
		os.Exit(0)
	}
	reader := bufio.NewReader(os.Stdin)
	var input string
	var status1, status2 bool = false, false
	for i := 0; ; i++ {
		s, _ := reader.ReadString('\n')
		input = input + s
		if !setFlags.IsSetL {
			flags = CheckFlags(false, f, len(input), setFlags)
		} else if i == 0 {
			flags = CheckFlags(false, f, len(input), setFlags)
		}
		if len(input)-flags.S < flags.C && len(input)-flags.S < flags.L {
			continue
		} else {
			if len(input)-flags.S >= flags.C {
				status1 = true
			}
			if len(input)-flags.S > flags.L {
				status2 = true
			}
			if len(input)-flags.S == flags.L {
				if setFlags.IsSetL {
					status2 = true
				} else {
					status2 = false
				}
			}
		}
		if status1 && status2 {
			inputParse([]byte(input[flags.S:flags.L+flags.S]), offset, flags, flags.L)
			break

		} else if status1 {
			for {
				inputParse([]byte(input[flags.S:flags.C+flags.S]), offset, flags, flags.C)
				input = input[flags.C:]
				flags.L = flags.L - flags.C
				offset += 1
				if flags.L < flags.C || len(input) < flags.C {
					break
				}
			}
			status1, status2 = false, false
		} else {
			inputParse([]byte(input[flags.S:flags.L+flags.S]), offset, flags, flags.L)
			break
		}
	}
}
func processFile(fileName string, f *Flags, setFlags *IsSetFlags) {
	var flags *ParsedFlags
	var length int = 0
	file, err := os.Open(fileName)
	if f.Revert {
		revert(file)
		os.Exit(0)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdxxd: %v: No such file or directory\n", fileName)
		os.Exit(1)
	}
	fileStat, _ := file.Stat()
	fileSize := fileStat.Size()

	flags = CheckFlags(true, f, int(fileSize), setFlags)
	defer file.Close()

	buffer := make([]byte, size(flags.C))
	offset := 0
	file.Seek(int64(flags.S), 0)
	for {
		n, err := file.Read(buffer)
		if err != nil {
			break
		}
		if flags.L < n {
			length = flags.L
		} else {
			length = n
			flags.L = flags.L - n
		}
		inputParse(buffer[:length], offset, flags, length)
		if length < size(flags.C) {
			break
		}
		offset += 1
	}
}
func Driver() {
	f, setFlags, args := NewFlags()
	if len(args) == 0 {
		processStdIn(f, setFlags)
	} else {
		processFile(args[0], f, setFlags)
	}
}
