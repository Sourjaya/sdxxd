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
	Endian    bool
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
	IsSetG bool
	IsSetL bool
	IsSetC bool
	IsSetS bool
}

type Result struct {
	Offset int
	Chunk  string
}

func NewFlags() (*Flags, *IsSetFlags, []string) {
	flags := new(Flags)
	setFlags := &IsSetFlags{}
	flag.BoolVarP(&flags.Endian, "little-endian", "e", false, "little-endian")
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
func inputParse(s []byte, offset int, f *ParsedFlags, length int) string {
	buffer := byteToHex(s, f.C)
	return dumpHex(offset, length, f, buffer, s)
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
func dumpHex(offset, length int, f *ParsedFlags, stringBuffer string, buffer []byte) (resultString string) {
	i, rowCount, groupCount := 0, 0, 0
	var groupBuffer string
	for i < length*2 {
		if !f.IsFile {
			resultString += fmt.Sprintf("%08x: ", (offset*f.C + f.C*rowCount + f.S))
		} else {
			resultString += fmt.Sprintf("%08x: ", (offset*size(f.C) + f.C*rowCount + f.S))
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
			resultString += fmt.Sprintf("%s ", groupBuffer)
			groupCount += 1

		}
		var originalBuffer string
		if (f.C * (rowCount + 1)) > len(buffer) {
			originalBuffer = bytesToString(buffer[(f.C * rowCount):])
		} else {
			originalBuffer = bytesToString(buffer[(f.C * rowCount):(f.C * (rowCount + 1))])
		}
		resultString += fmt.Sprintf(" %v\n", originalBuffer)
		i += f.C * 2
		rowCount += 1
	}
	return resultString
}

func checkFlags(isFile bool, f *Flags, size int, setFlags *IsSetFlags) (*ParsedFlags, int) {
	flag := &ParsedFlags{}
	flag.R = f.Revert
	flag.E = f.Endian
	flag.IsFile = isFile

	var res int64
	var err error
	flag.L = size
	if setFlags.IsSetL {
		if res, err = NumberParse(f.Length); err != nil || res == 0 {
			return flag, 1
		}
		flag.L = int(res)
		if res < 0 || (flag.L > size && isFile) {
			flag.L = size
		}
	}
	if setFlags.IsSetG {
		if res, err = NumberParse(f.GroupSize); err != nil || res == 0 {
			flag.G = 16
		} else if res < 0 {
			if f.Endian {
				flag.G = 4
			} else {
				flag.G = 2
			}
		} else if res > 0 {
			flag.G = int(res)
			if f.Endian && res&(res-1) != 0 {
				fmt.Println("sdxxd: number of octets per group must be a power of 2 with -e.")
				return flag, 1
			}
		}
	} else if f.Endian {
		flag.G = 4
	} else {
		flag.G = 2
	}

	flag.C = 16
	if setFlags.IsSetC {
		if res, err := NumberParse(f.Columns); err != nil {
			return flag, 1
		} else {
			flag.C = int(res)
		}
	}
	if setFlags.IsSetS {
		if f.Seek == "-0" && !isFile {
			fmt.Fprintln(os.Stderr, "sdxxd: Sorry, cannnot seek.")
			return flag, 4
		} else if f.Seek == "-0" && isFile {
			flag.S = size
		} else if res, err := NumberParse(f.Seek); err == nil {
			flag.S = int(res)
			if res < 0 {
				flag.S = size + int(res)
			}
		}
	}
	return flag, 0
}
func size(cols int) int {
	div := 2048 / cols
	if 2048%cols != 0 {
		return (div + 1) * cols
	}
	return div * cols
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
				return errors.New("error while decoding")
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
			return errors.New("error while decoding")
		}
		os.Stdout.Write(decodedString)
	}
	return nil
}
func processStdIn(f *Flags, setFlags *IsSetFlags) int {
	offset, status := 0, 0
	var flags *ParsedFlags
	scanner := bufio.NewScanner(os.Stdin)
	if f.Revert {
		//fmt.Println("Starting revert")
		err := revert(scanner)
		if err != nil {
			return 1
		}
		return 0
	}
	var i = 0
	var input string
	var status1, status2 bool = false, false
	var ch chan int
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		s := scanner.Text()
		input = input + s
		if !setFlags.IsSetL || i == 0 {
			flags, status = checkFlags(false, f, len(input), setFlags)
			if status != 0 {
				close(ch)
				return status
			}
		}
		l1 := len(input) - flags.S
		if l1 < flags.C && l1 < flags.L {
			continue
		} else {
			status1 = l1 >= flags.C
			status2 = l1 > flags.L || l1 == flags.L && setFlags.IsSetL
		}
		if (status1 && status2) || !status1 {
			inputParse([]byte(input[flags.S:flags.L+flags.S]), offset, flags, flags.L)
			close(ch)
			return 0
		}
		if status1 {
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
		}
		i += 1
	}

	for i := 0; i < offset; i++ {
		<-ch
	}
	close(ch)
	return 0
}
func processFile(fileName string, f *Flags, setFlags *IsSetFlags) int {
	var length int = 0
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdxxd: %v: No such file or directory\n", fileName)
		return 2
	}
	if f.Revert {
		//fmt.Println("Starting revert")
		err := revert(file)
		if err != nil {
			return 2
		}
		return 0
	}
	fileStat, err := file.Stat()
	if err != nil {
		return 2
	}
	fileSize := fileStat.Size()
	flags, status := checkFlags(true, f, int(fileSize), setFlags)
	if status != 0 {
		return status
	}
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
		fmt.Print(inputParse(buffer[:length], offset, flags, length))
		if length < size(flags.C) {
			break
		}
		offset += 1
	}
	return 0
}
func Driver() int {
	f, setFlags, args := NewFlags()
	if len(args) == 0 {
		return processStdIn(f, setFlags)
	}
	return processFile(args[0], f, setFlags)
}
