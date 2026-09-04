package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println(
			`Error: This command requires two arguments.
Usage: cp-plugin src dst`)
		os.Exit(1)
	}
	src, dst := os.Args[1], os.Args[2]
	fmt.Printf("Copying %s to %s ...  ", src, dst)
	srcFile, err := os.Open(src)
	if err != nil {
		exitOnError("open source file %s: %v", src, err)
	}
	defer srcFile.Close()
	if _, err := os.Stat(dst); errors.Is(err, os.ErrNotExist) {
		_, err = os.Create(dst)
		if err != nil {
			exitOnError("create destination file %s: %v", dst, err)
		}
	}
	dstFile, err := os.OpenFile(dst, os.O_WRONLY, 0755)
	if err != nil {
		exitOnError("open destination file %s for writing: %v", dst, err)
	}
	defer dstFile.Close()
	buf := make([]byte, 1024*128)
	_, err = io.CopyBuffer(dstFile, srcFile, buf)
	if err != nil {
		exitOnError("copy %s to %s: %v", src, dst, err)
	}
	os.Chmod(dst, 0755)
	fmt.Println("done.")
}

func exitOnError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: failed to "+format+"\n", args...)
	os.Exit(1)
}
