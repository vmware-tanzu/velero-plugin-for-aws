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
		panic(err)
	}
	defer srcFile.Close()
	if _, err := os.Stat(dst); errors.Is(err, os.ErrNotExist) {
		_, err = os.Create(dst)
		if err != nil {
			panic(err)
		}
	}
	dstFile, err := os.OpenFile(dst, os.O_WRONLY, 0755)
	if err != nil {
		panic(err)
	}
	defer dstFile.Close()
	buf := make([]byte, 1024*128)
	_, err = io.CopyBuffer(dstFile, srcFile, buf)
	if err != nil {
		panic(err)
	}
	os.Chmod(dst, 0755)
	fmt.Println("done.")
}
