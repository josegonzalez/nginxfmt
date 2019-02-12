package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var isWord = regexp.MustCompile(`^[a-zA-Z_]+$`).MatchString

func main() {
	override := flag.Bool("i", false, "override origin file")
	file := flag.String("f", "", "format nginx conf file path")
	dir := flag.String("d", "", "nginx conf dir")
	ext := flag.String("e", ".conf", "nginx conf extension")
	minify := flag.Bool("m", false, "minify program to reduce its size")

	flag.Parse()

	if *dir != "" {
		filepath.Walk(*dir, func(path string, info os.FileInfo, e error) error {
			if strings.HasSuffix(path, *ext) {
				fmtFile(path, *override, *minify)
			}
			return nil
		})
	} else if *file != "" {
		fmtFile(*file, *override, *minify)
	} else {
		flag.Usage()
	}
}

func isComment(l string) bool {
	return strings.HasPrefix(l, "#")
}

func isBlockStart(l string) bool {
	return strings.HasSuffix(l, "{")
}

func isBlockEnd(l string) bool {
	return l == "}"
}

func isNewLine(l string) bool {
	return l == ""
}

func getType(l string) string {
	if isComment(l) {
		return "comment"
	}
	if isBlockStart(l) {
		return "block_start"
	}
	if isBlockEnd(l) {
		return "block_end"
	}
	if isNewLine(l) {
		return "newline"
	}
	return "directive"
}

func writeString(buf *bytes.Buffer, str string, indent int, newlineStart bool, newlineEnd bool) bool {
	if newlineStart {
		buf.WriteString("\n")
	}
	buf.WriteString(strings.Repeat("  ", indent))
	buf.WriteString(str)
	if newlineEnd {
		buf.WriteString("\n")
	}
	return true
}

func flushDirectives(buf *bytes.Buffer, directives []string, indent int, maxLength int) int {
	for _, directive := range directives {
		parts := strings.SplitN(directive, " ", 2)
		if len(parts) != 1 && isWord(parts[0]) {
			maxLength = max(maxLength, len(parts[0]))
		}
	}

	for _, directive := range directives {
		parts := strings.SplitN(directive, " ", 2)
		if len(parts) == 2 && isWord(parts[0]) {
			directivePrefix := rightPad(parts[0], maxLength, " ")
			directiveSuffix := strings.TrimSpace(parts[1])
			writeString(buf, fmt.Sprintf("%s %s", directivePrefix, directiveSuffix), indent, false, true)
		} else {
			writeString(buf, directive, indent, false, true)
		}
	}

	return maxLength
}

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func rightPad(str string, length int, pad string) string {
	return str + times(pad, length-len(str))
}

func times(str string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(str, n)
}

func fmtFile(file string, override bool, minify bool) {
	f, err := os.Open(file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rd := bufio.NewReader(f)
	var buf bytes.Buffer
	var indent = 0
	var prevType = "nil"
	var started = false
	var directives = []string{}
	var maxLength = 0

	for {
		l, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		l = strings.TrimSpace(l)

		currType := getType(l)

		if minify && (currType == "comment" || currType == "newline") {
			continue
		}

		if currType != "directive" {
			maxLength = flushDirectives(&buf, directives, indent, maxLength)
			directives = []string{}
		}

		switch currType {
		case "comment":
			newlineStart := prevType != "comment" && prevType != "block_start"
			writeString(&buf, "# "+strings.TrimSpace(l[1:]), indent, newlineStart, true)
		case "block_start":
			maxLength = 0
			newlineStart := prevType != "comment" && started
			writeString(&buf, l, indent, newlineStart, true)
			indent++
		case "block_end":
			maxLength = 0
			indent--
			writeString(&buf, l, indent, false, true)
		case "directive":
			directives = append(directives, l)
		case "newline":
		}

		prevType = currType
		if currType != "newline" {
			started = true
		}
	}

	if !override {
		fmt.Println(buf.String())
	} else {
		fmt.Println("format", file, "OK")
		ioutil.WriteFile(file, buf.Bytes(), 0666)
	}
}
