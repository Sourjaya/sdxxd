// Implementation of command line tool `xxd`.

// Package xxd implements the logic behind converting byte stream to hexadecimal dump.
// It can also convert a hexadecimal dump back to its original binary form.
// This functionality is otherwise provide by linux command line tool xxd.
package xxd

// import other packages
import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	pflag "github.com/spf13/pflag"
)

const (
	CSIZE = 4096
	SIZE  = 2048
)

/*
Struct containing flag values as entered in the terminal.

	e stands for little-endian output

	g stands for Group-Size

	l stands for the no of bytes to convert

	c stands for columns to print each line

	s stands for seek

	r stands for reverting from hex dump to original file
*/
type Flags struct {
	GroupSize string
	Length    string
	Columns   string
	Seek      string
	Revert    bool
	Endian    bool
}

// Struct containing parsed flag values from original values
type ParsedFlags struct {
	G      int
	L      int
	S      int
	C      int
	R      bool
	IsFile bool
	E      bool
}

// Struct to indicate whether a particular flag was used as options
type IsSetFlags struct {
	IsSetG bool
	IsSetL bool
	IsSetC bool
	IsSetS bool
}

// Function to parse flags from the command line.
func NewFlags() (*Flags, *IsSetFlags, []string) {
	flags := new(Flags)
	setFlags := &IsSetFlags{}
	// define the flags with their default values
	pflag.BoolVarP(&flags.Endian, "little-endian", "e", false, "little-endian")
	pflag.StringVarP(&flags.GroupSize, "group-size", "g", "2", "group-size")
	pflag.StringVarP(&flags.Length, "length", "l", "-1", "length")
	pflag.StringVarP(&flags.Columns, "cols", "c", "16", "columns")
	pflag.StringVarP(&flags.Seek, "seek", "s", "0", "seek")
	pflag.BoolVarP(&flags.Revert, "revert", "r", false, "revert")
	// parse the flags
	pflag.Parse()
	// visit the flags in lexographical order and check if flags be visited or not
	pflag.Visit(func(f *pflag.Flag) {
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

	args := pflag.Args()
	// return the flag values along with the setFlags status and the arguments list
	return flags, setFlags, args
}

// Function to parse number from a string using regular expression
func numberParse(input string) (res int64, err error) {
	// regular expression
	re := regexp.MustCompile(`-?0[xX][0-9a-fA-F]+|-\b0[0-7]*\b|-\b[1-9][0-9]*\b|0[xX][0-9a-fA-F]+|\b0[0-7]*\b|\b[1-9][0-9]*\b`)
	// Find the match
	s := re.FindString(input)
	// if a certain match is found convert into decimal, octal or hexadecimal and return. else return 0.
	if s != "" {
		return strconv.ParseInt(s, 0, 64)
	}

	return 0, nil
}

// Function to parse the input stream of bytes.
func InputParse(s []byte, offset int, f *ParsedFlags, length int) string {
	// convert byte slice to hex string
	buffer := byteToHex(s, f.C)
	// function to generate hex dump output string
	return dumpHex(offset, length, f, buffer, s)
}

// Function to reverse a string
// input: The input hex string to be reversed.
// Returns the reversed hex string.
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

// Function to convert a byte slice to a hex string with specified grouping.
// byteBuffer: The input byte slice to be converted.
// count: The number of bytes per group.
// Returns the hex string representation of the byte slice.
func byteToHex(byteBuffer []byte, count int) string {
	// encode byte slice to string
	encodedString := hex.EncodeToString(byteBuffer)
	// add extra whitespaces
	for i := 0; i < (count-(len(byteBuffer)%count))*2; i++ {
		encodedString = fmt.Sprint(encodedString, " ")
	}

	return encodedString
}

// input: The input byte slice to be converted.
// Returns the string representation of the byte slice.
func bytesToString(input []byte) string {
	output := make([]byte, len(input))
	// convert ASCII byte slice to its equivalent character string
	for i, b := range input {
		if b < 0x20 || b > 0x7e {
			output[i] = '.'
		} else {
			output[i] = b
		}
	}

	return string(output)
}

// Function to generate a hexadecimal dump output string.
// offset: The starting offset value. this helps in line indexing.
// length: The length of the buffer.
// f: ParsedFlags containing information about flag values.
// stringBuffer: The hex string buffer.
// buffer: The byte buffer.
// Returns the hexadecimal dump output string.
func dumpHex(offset, length int, f *ParsedFlags, stringBuffer string, buffer []byte) (resultString string) {
	i, rowCount, groupCount := 0, 0, 0

	var groupBuffer string

	for i < length*2 {
		// print the 8 byte offset
		if !f.IsFile {
			resultString += fmt.Sprintf("%08x: ", (offset*f.C + f.C*rowCount + f.S))
		} else {
			resultString += fmt.Sprintf("%08x: ", (offset*size(f.C) + f.C*rowCount + f.S))
		}

		groupCount = 1
		// print the grouped hex bytes for each line
		for j := 0; j < f.C*2; j += f.G * 2 {
			if groupCount*f.G*2 > f.C*2 {
				groupBuffer = stringBuffer[i+j : i+(f.C*2)]
			} else {
				groupBuffer = stringBuffer[i+j : i+(groupCount*f.G*2)]
			}
			// reverse the string if e flag is provided
			if f.E {
				groupBuffer = reverseString(groupBuffer)
			}

			resultString += fmt.Sprintf("%s ", groupBuffer)
			groupCount += 1
		}

		var originalBuffer string
		// print the original character bytes for the line
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

// Function to check for validity of flag values entered.
func checkFlags(isFile bool, f *Flags, size int, setFlags *IsSetFlags) (*ParsedFlags, int) {
	flags := &ParsedFlags{}
	flags.R = f.Revert
	flags.E = f.Endian
	flags.IsFile = isFile

	var res int64

	var err error

	flags.L = size
	// check for l flag validity and set correct value of l parameter
	if setFlags.IsSetL {
		if res, err = numberParse(f.Length); err != nil || res == 0 {
			return flags, 1
		}

		flags.L = int(res)
		if res < 0 || (flags.L > size && isFile) {
			flags.L = size
		}
	}
	// check for g flag validity and set correct value of g parameter
	if setFlags.IsSetG {
		if res, err = numberParse(f.GroupSize); err != nil || res == 0 {
			flags.G = 16
		} else if res < 0 {
			if f.Endian {
				flags.G = 4
			} else {
				flags.G = 2
			}
		} else if res > 0 {
			flags.G = int(res)

			if f.Endian && res&(res-1) != 0 {
				fmt.Println("sdxxd: number of octets per group must be a power of 2 with -e.")
				return flags, 1
			}
		}
	} else if f.Endian {
		flags.G = 4
	} else {
		flags.G = 2
	}

	flags.C = 16
	// check for c flag validity and set correct value of c parameter
	if setFlags.IsSetC {
		var res int64

		if res, err = numberParse(f.Columns); err != nil {
			return flags, 1
		}

		flags.C = int(res)
	}
	// check for s flag validity and set correct value of s parameter
	if setFlags.IsSetS {
		if f.Seek == "-0" && !isFile || (len(f.Seek) > 1 && f.Seek[:2] == "+-") {
			fmt.Fprintln(os.Stderr, "sdxxd: Sorry, cannot seek.")
			return flags, 4
		} else if f.Seek == "-0" && isFile {
			flags.S = size
		} else if res, err := numberParse(f.Seek); err == nil {
			flags.S = int(res)
			if res < 0 {
				flags.S = size + int(res)
			}
		}
	}

	return flags, 0
}

// calculate size of chunk to read for each iteration
func size(cols int) int {
	div := SIZE / cols
	if SIZE%cols != 0 {
		return (div + 1) * cols
	}

	return div * cols
}

// Helper function to trim the spaces from a line
func trimBytes(s string) string {
	words := strings.Fields(s)

	return strings.Join(words, "")
}

// Function to convert (or patch) hexdump into binary.
func revert(input any) error {
	defer func() any {
		err := recover()
		return err
	}()

	var str string
	// switch case for two types of inputs: file and standard input.
	switch v := input.(type) {
	case *os.File:
		scanner := bufio.NewScanner(v)

		for {
			for scanner.Scan() {
				// for each scanned line of hex dump trim the spaces and split the hex bytes from the offset and the original file string.
				field := trimBytes(strings.TrimSpace(strings.Split(strings.Split(scanner.Text(), ":")[1], "  ")[0]))
				str += field

				if len(str) > CSIZE {
					break
				}
			}
			// decode the string to slice of bytes
			decodedString, err := hex.DecodeString(str)
			if err != nil {
				return errors.New("error while decoding")
			}
			// write the slice of bytes to standard output
			os.Stdout.Write(decodedString)

			if len(str) < CSIZE {
				break
			}

			str = ""
		}
	case *bufio.Scanner:
		v.Split(bufio.ScanLines) // set the split function which in this case is bufio.ScanLines which tells bufio.Scanner to scan one line at a time

		for v.Scan() {
			line := v.Text()
			// for each scanned line of hex dump trim the spaces and split the hex bytes from the offset and the original file string.
			field := trimBytes(strings.TrimSpace(strings.Split(strings.Split(line, ":")[1], "  ")[0]))
			str += field
		}
		// decode the string to slice of bytes
		decodedString, err := hex.DecodeString(str)
		if err != nil {
			return errors.New("error while decoding")
		}
		// write the slice of bytes to standard output
		os.Stdout.Write(decodedString)
	}

	return nil
}

// Function to convert input from standard input to hex dump
func processStdIn(f *Flags, setFlags *IsSetFlags) int {
	var flags *ParsedFlags

	offset, status := 0, 0

	scanner := bufio.NewScanner(os.Stdin)
	// if r flag is set, call revert function and return
	if f.Revert {
		err := revert(scanner)
		if err != nil {
			return 1
		}

		return 0
	}

	if setFlags.IsSetS && ((len(f.Seek) > 1 && f.Seek[:2] == "+-") || f.Seek[:1] == "-") {
		fmt.Fprintln(os.Stderr, "sdxxd: Sorry, cannot seek.")
		return 4
	}
	// if r flag is not set, read from standard input and show the hex dump. The program will continue until interrupt.
	var input string

	status1, status2 := false, false
	i := 0

	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		s := scanner.Text()
		input += s
		// check for flag validity and set proper values
		if !setFlags.IsSetL || i == 0 {
			flags, status = checkFlags(false, f, len(input), setFlags)
			if status != 0 {
				return status
			}
		}
		// checking for different edge cases
		l1 := len(input) - flags.S
		if l1 < flags.C && ((l1 <= flags.L && !setFlags.IsSetL) || l1 < flags.L) {
			continue
		}

		status1 = l1 >= flags.C
		status2 = l1 > flags.L || l1 == flags.L && setFlags.IsSetL

		if (status1 && status2) || !status1 {
			fmt.Print(InputParse([]byte(input[flags.S:flags.L+flags.S]), offset, flags, flags.L))
			return 0
		}

		if status1 {
			for {
				fmt.Print(InputParse([]byte(input[flags.S:flags.C+flags.S]), offset, flags, flags.C))
				input = input[flags.C:]
				flags.L -= flags.C
				offset += 1

				if flags.L < flags.C || len(input) < flags.C {
					break
				}
			}
			status1, status2 = false, false
		}
		i += 1
	}

	return 0
}

// Function to convert input from file to hex dump
func processFile(fileName string, f *Flags, setFlags *IsSetFlags) int {
	length := 0
	file, err := os.Open(fileName)
	// check if file is present or not
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdxxd: %v: No such file or directory\n", fileName)
		return 2
	}
	// if r flag is set, call revert function and return
	if f.Revert {
		err = revert(file)
		if err != nil {
			return 2
		}

		return 0
	}
	// if r flag is not set, the contents of the file will be converted to a hex dump.
	fileStat, err := file.Stat()
	if err != nil {
		return 2
	}

	fileSize := fileStat.Size()
	// check for flag validity and set proper values
	flags, status := checkFlags(true, f, int(fileSize), setFlags)
	if status != 0 {
		return status
	}

	defer file.Close()

	buffer := make([]byte, size(flags.C))
	offset := 0

	if _, err = file.Seek(int64(flags.S), 0); err != nil {
		return 2
	}

	// loop until all chunks of data are parsed
	for {
		n, err := file.Read(buffer)
		if err != nil {
			break
		}
		// set the proper length and update the no of bytes left to print, after each iteration
		if flags.L < n {
			length = flags.L
		} else {
			length = n
			flags.L -= n
		}
		// parsing input
		fmt.Print(InputParse(buffer[:length], offset, flags, length))

		if length < size(flags.C) {
			break
		}
		offset += 1
	}

	return 0
}

// Driver function to use the functionalities of this package
func Driver() int {
	f, setFlags, args := NewFlags()
	// if no file name is provided read from standard input
	if len(args) == 0 || args[0] == "-" {
		return processStdIn(f, setFlags)
	}

	return processFile(args[0], f, setFlags)
}
