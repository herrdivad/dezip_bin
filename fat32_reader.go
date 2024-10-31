package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/diskfs/go-diskfs"            // MIT License: Copyright (c) 2017 Avi Deitcher
	"github.com/diskfs/go-diskfs/filesystem" // MIT License Copyright: (c) 2017 Avi Deitcher
)

/**************************************************************************
 * File:        <fat32_reader.go>
 * Author:      David Herrmann
 * Email:       <david.herrmann@kit.edu>
 * Created:     <31.10.2024>
 *
 * Project:     <dezip_bin>
 *
 * Description: A GOLANG script to extract latest file(s) from a PIUSB.bin file
 *
 * ------------------------------------------------------------------------
 * License:
 * MIT License
 *
 * Copyright (c) <2024> David Herrmann
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 * -----------------------------------------------------------------------
 */

var targetDir string = "./target/" // Update this path to your target (output) directory
var currentPath = "/"

func main() {
	// Path to the FAT32 image and target directory
	imagePath := "./piusb.bin" // Update this path to your .bin file

	dt := time.Now()
	fmt.Println("Current date and time is: ", dt.String())

	device := ""

	if len(os.Args) > 1 {
		device = os.Args[1] // "device" is only a specific case, you could almost target any subfolder inside the bin-file.
		fmt.Println("This script is made for the ", device, " device")
	} else {
		fmt.Println("This script is made for the root directory of bin file")
	}

	// Create target directory if it doesn't exist
	err := os.MkdirAll(targetDir, 0755)
	if err != nil {
		log.Fatalf("Error creating target directory: %v\n", err)
	}

	// Open the disk image
	disk, err := diskfs.Open(imagePath)
	if err != nil {
		fmt.Printf("Error opening disk image: %v\n", err)
		return
	}
	defer disk.Close()

	// Get the first partition
	mainFS, err := disk.GetFilesystem(0) // Access the first partition directly
	if err != nil {
		log.Panic(err)
	}

	// List files in the root directory
	// This should list everything
	files := readPath(mainFS, currentPath)

	// Going deeper into device results folder
	if len(os.Args) > 1 {
		currentPath = "/" + device + "/results" // remove or rename "results" if you are not targeting this kind of folders
		files = readPath(mainFS, currentPath)
	}

	sort.Slice(files, func(i, j int) bool {
		// Always place "System Volume Information" last
		if files[i].Name() == "System Volume Information" {
			return false
		}
		if files[j].Name() == "System Volume Information" {
			return true
		}
		// Sort by modification time for other files (newest first)
		return files[i].ModTime().After(files[j].ModTime())
	})

	fmt.Println(currentPath, "directory contents sorted by Modtime:")
	for _, file := range files {
		fmt.Printf("%s - %v time\n", file.Name(), file.ModTime())
	}

	for i, file := range files {

		fileName := file.Name()

		if i == 0 {

			if !readAndExtractFile(fileName, mainFS) {
				fmt.Println("Operation failed")
				return
			}

			fmt.Println("Operation succeeded")
		}
		if i > 0 {

			fmt.Println("Looking for more files with same name but different extension ...")

			if fileNameWithoutExt(fileName) != fileNameWithoutExt(files[0].Name()) {
				fmt.Println("No more files with same name")
				break

			} else {
				if !readAndExtractFile(fileName, mainFS) {
					fmt.Println("Operation failed")
					return
				}
			}
		}
	}

	fmt.Println("EOC reached")

}

func readPath(myFS filesystem.FileSystem, mypath string) []fs.FileInfo {

	fmt.Println(
		mypath, "directory contents (only Files are further used,  \n to search result folders of an instrument, please use CLI-Arguments like SMP50):")

	files, err := myFS.ReadDir(mypath)
	if err != nil {
		log.Panic(err)
	}

	// Slice to store only the files (not directories)
	var fileList []fs.FileInfo

	for _, file := range files {
		// Skip directories
		if !(file.IsDir()) {
			// Add only files to the list
			fileList = append(fileList, file)
		}

		// Print file details
		fmt.Printf("%s - %d bytes\n", file.Name(), file.Size())
	}
	return fileList
}

func readAndExtractFile(fileName string, myFS filesystem.FileSystem) bool {
	fmt.Printf("The latest file is %s \n", fileName)

	// Construct the absolute path using the correct separator for your operating system
	absolutePath := currentPath + "/" + string(fileName) // -> filepath.Join("/", fileName) will !not! work, maybe Windows or FAT32 !!!!!!!!
	fmt.Printf("The absolute path is %s \n", absolutePath)

	// Attempt to open the file using the absolute path
	fileEntry, err := myFS.OpenFile(string(absolutePath), 0) // Use absolute path
	if err != nil {
		fmt.Printf("Error opening file %s: %v\n", absolutePath, err)
		return false
	}

	defer fileEntry.Close()

	// Create the target file in the target directory
	outputPath := filepath.Join(targetDir, fileName)
	outputFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return false
	}
	defer outputFile.Close()

	// Copy the file contents to the target file
	_, err = io.Copy(outputFile, fileEntry)
	if err != nil {
		fmt.Printf("Error copying file contents: %v\n", err)
		return false
	}

	fmt.Printf("Extracted %s to %s\n", fileName, outputPath)
	return true
}

func fileNameWithoutExt(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}
