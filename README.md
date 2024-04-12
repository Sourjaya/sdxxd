# xxd
`xxd` is a Go package that provides functionality similar to the **xxd** command-line tool in Linux. It allows converting byte streams to hexadecimal dumps and vice versa.

---

# Installation
You can install the xxd package using the go get command:
```bash
go get github.com/your-username/xxd
```

---

# Usage
After installation, you can use the xxd package in your Go programs. Here's an example of how to use it:

```go
package main

import (
	"log"
	"github.com/Sourjaya/sdxxd/xxd"
)

func main() {
	// Call the Driver function to utilize the functionalities of the xxd package
	statusCode := xxd.Driver()
	// Checking for error code
	if statusCode != 0 {
		log.Println("Error:", statusCode)
	}
}
```
If you want to test it out in the terminal, first build the go binary using `go build`, and then use the binary file just like any other command.
```bash
# ./sdxxd <filename>
# ./sdxxd <filepath>
# ./sdxxd -help
./sdxxd file.txt
```
---

# Command Line Flags
The xxd package supports the following command-line flags:


| Flags       | Description                                          |
|-------------|------------------------------------------------------|
| -e          | Output in little-endian format.                      |
| -g          | Specify the number of bytes per group.               |
| -l          | Specify the no of bytes to dump from the input.      |
| -c          | Specify the number of columns to print each line.    |
| -s          | Specify the offset to seek to before reading.        |
| -r          | Revert from hexadecimal dump to original binary form.|

---

# Code Reference
- #### `func Driver() int`
This function is the entry point for utilizing the functionalities provided by the xxd package. It parses command-line flags and processes input from either standard input or files.

- #### `func NewFlags() (*Flags, *IsSetFlags, []string)`
This function initializes and parses command-line flags.

- #### `func numberParse(input string) (int64, error)`
This function parses numbers from strings using regular expressions.

- #### `func (flags *ParsedFlags) InputParse(s []byte, offset int, length int) string`
This function parses the input stream of bytes and generates a hexadecimal dump output string.

- #### `func (f *Flags) checkFlags(isFile bool, size int, setFlags *IsSetFlags) (*ParsedFlags, int)`
This function checks the validity of flag values entered.

- #### `func revert(input any) error`
This function converts (or patches) a hex dump into binary.

- #### `func (f *Flags) processStdIn(setFlags *IsSetFlags) int `
This function processes input from standard input and converts it to a hex dump.

- #### `func (f *Flags) processFile(fileName string, setFlags *IsSetFlags) int`
This function processes input from a file and converts it to a hex dump.

- #### `func reverseString(input string) string`
This function is used to reverse a string
- #### `func byteToHex(byteBuffer []byte, count int) string`
This function is used to convert a byte slice to a hex string with specified grouping.
- #### `func bytesToString(input []byte) string`
This function converts ASCII byte slice to its equivalent character string
- #### `func (flags *ParsedFlags) dumpHex(offset, length int, stringBuffer string, buffer []byte) (resultString string)`
This function generates a hexadecimal dump output string.

---

# TODO
Enable parallel read and processing of bytes. Add other flag functionalities.



