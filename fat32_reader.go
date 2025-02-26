package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	//"github.com/herrdivad/go-diskfs"
	//"github.com/herrdivad/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs"            // MIT License: Copyright (c) 2017 Avi Deitcher
	"github.com/diskfs/go-diskfs/filesystem" // MIT License: Copyright (c) 2017 Avi Deitcher
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

type FileWithPath struct {
	ItsFileInfo fs.FileInfo
	ItsPath     string
}

type FSFileWithPath struct {
	FileSystem filesystem.FileSystem
	FilePath   string
}

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
	files := readPath(mainFS, currentPath, false)

	/*
		err = mainFS.RemoveFile("/DHM20241004.pdf")
		if err != nil {
			log.Panic(err)
		}


		fmt.Println("\n After remove test \n")

		files = readPath(mainFS, currentPath)

	*/

	// Going deeper into device results folder
	if len(os.Args) > 1 {
		if device == "all" || device == "--all" || device == "-all" || device == "-a" || device == "each" {
			files = readPath(mainFS, currentPath, true)
		} else {
			currentPath = "/" + device + "/results" // remove or rename "results" if you are not targeting this kind of folders
			files = readPath(mainFS, currentPath, false)
		}

	}

	sort.Slice(files, func(i, j int) bool {
		// Always place "System Volume Information" last
		if files[i].ItsFileInfo.Name() == "System Volume Information" {
			return false
		}
		if files[j].ItsFileInfo.Name() == "System Volume Information" {
			return true
		}
		// Sort by modification time for other files (newest first)
		return files[i].ItsFileInfo.ModTime().After(files[j].ItsFileInfo.ModTime())
	})

	fmt.Println(currentPath, "directory contents sorted by Modtime:")
	for _, file := range files {
		fmt.Printf("%s: %s - %v time\n", file.ItsPath, file.ItsFileInfo.Name(), file.ItsFileInfo.ModTime())
	}

	shouldReturn := loopThroughFilesToExtract(files, mainFS)
	if shouldReturn {
		fmt.Println("ERROR during Loop through Files")
		return
	}

	FileToRemove := FSFileWithPath{
		FileSystem: mainFS,
		FilePath:   "/my-go-project",
	}
	fmt.Println(tidyUp(FileToRemove))
	fmt.Println(tidyUp(targetDir + "uvvis_measurement.txt"))

	fmt.Println("EOC reached")

}

func loopThroughFilesToExtract(files []FileWithPath, myFS filesystem.FileSystem) bool {

	rangeArgument := "--latest"
	if len(os.Args) > 2 {
		rangeArgument = os.Args[2]
	}

	for i, file := range files {

		fileName := file.ItsFileInfo.Name()

		if i == 0 {

			if !readAndExtractFile(file, myFS, true) {
				fmt.Println("Operation failed")
				return true
			}

			fmt.Println("Operation succeeded")
		}
		if i > 0 {

			switch rangeArgument {
			case "--latest":
				return false
			case "--latestSame":
				fmt.Println("Looking for more files with same name but different extension ...")

				if fileNameWithoutExt(fileName) != fileNameWithoutExt(files[0].ItsFileInfo.Name()) {
					fmt.Println("No more files with same name")
					return false

				} else {
					if !readAndExtractFile(file, myFS, true) {
						fmt.Println("Operation failed")
						return true
					}
				}
			case "--all":
				if !readAndExtractFile(file, myFS, true) {
					fmt.Println("Operation failed")
					return true
				}

				fmt.Println("Operation succeeded")
			default:
				panic("Unexpected case: this should not happen \n Please only use this cases for the 2nd argument --latest, --latestSame, --all")
			}

		}
		var shouldTidyUp bool
		if len(os.Args) > 3 {
			shouldTidyUp, _ = strconv.ParseBool(os.Args[3])
		}
		if shouldTidyUp {
			FileToRemove := FSFileWithPath{
				FileSystem: myFS,
				FilePath:   file.ItsPath + "/" + fileName,
			}
			fmt.Println("I tried to delet", FileToRemove.FilePath, "The result is: ", tidyUp(FileToRemove))
		}
	}
	return false
}

func readPath(myFS filesystem.FileSystem, mypath string, recursive bool) []FileWithPath {

	fmt.Println(
		mypath, "directory contents (only Files are further used,  \n to search result folders of an instrument, please use CLI-Arguments like SMP50):")

	files, err := myFS.ReadDir(mypath)
	if err != nil {
		log.Panic(err)
	}

	var fileList []FileWithPath

	for _, file := range files {
		if file.Name() == "." || file.Name() == ".." || file.Name() == "System Volume Information" || file.Name() == "LOST.DIR" {
			continue
		}
		if file.IsDir() {
			// If recursive is enabled, scan subdirectory
			if recursive {
				fmt.Println("Going recursive...")
				newPath := mypath + "/" + file.Name()
				fmt.Printf("Reaching %s ...\n", newPath)
				subDirFiles := readPath(myFS, newPath, true) // Recursive call
				fileList = append(fileList, subDirFiles...)  // Merge results
			}
		} else {
			// Add file to the list
			fileList = append(fileList, FileWithPath{ItsFileInfo: file, ItsPath: mypath})
		}
	}

	// Print file details
	for _, file := range fileList {
		fmt.Printf("%s - %d bytes in path %s\n", file.ItsFileInfo.Name(), file.ItsFileInfo.Size(), file.ItsPath)
	}

	return fileList
}

func readAndExtractFile(file FileWithPath, myFS filesystem.FileSystem, check ...bool) bool {
	fileName := file.ItsFileInfo.Name()
	fmt.Printf("The latest file is %s \n", fileName)

	// Construct the absolute path using the correct separator for your operating system
	// absolutePath := currentPath + "/" + string(fileName) // -> filepath.Join("/", fileName) will !not! work, maybe Windows or FAT32 !!!!!!!!
	absolutePath := file.ItsPath + "/" + string(fileName)
	fmt.Printf("The absolute path is %s \n", absolutePath)

	// Attempt to open the file using the absolute path
	fileEntry, err := myFS.OpenFile(string(absolutePath), 0) // Use absolute path
	if err != nil {
		fmt.Println("Hier sollte ich aussteigen...")
		fmt.Printf("Error opening file %s: %v\n", absolutePath, err)
		return false
	}

	defer fileEntry.Close()

	outputPath := filepath.Join(targetDir, file.ItsPath, fileName)

	// Default value for check is false
	shouldCheckCS := false
	if len(check) > 0 {
		shouldCheckCS = check[0]
	}

	// Process logic based on `shouldCheck`
	if shouldCheckCS {
		var cs [2]string
		cs[0], err = getChecksum(fileEntry)
		if err != nil {
			fmt.Printf("Error getting CheckSum of file from bin-file: %v\n", err)
			return false
		}

		if fileExists(outputPath) {
			cs[1], err = getChecksum(outputPath)
			if err != nil {
				fmt.Printf("Error getting CheckSum of file from bin-file in target Dir: %v\n", err)
				return false
			}
		}

		fmt.Println("The CheckSum of both files are", cs)

		if cs[0] == cs[1] { // if the checksum of source and target file is identical, file should not be extracted and will be skipped
			fmt.Printf("Skipping extraction of %s to %s because the files have the same CheckSum\n", fileName, outputPath)
			return true
		}
	}

	// Ensure the directory path exists before creating the file
	if err := os.MkdirAll(filepath.Dir(outputPath), os.ModePerm); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return false
	}
	// Create the target file in the target directory
	outputFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return false
	}
	defer outputFile.Close()

	// Copy the file contents to the target file
	fmt.Println("Trying to copy content...")
	_, err = fileEntry.Seek(0, io.SeekStart) // Go to start of file (if CheckSum Calculater reached already EOF)
	if err != nil {
		fmt.Printf("Error resetting file pointer: %v\n", err)
		return false
	}
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

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false // File does not exist
	}
	return err == nil // Returns true if file exists, false if there's another error
}

// getChecksum computes the SHA256 checksum for a file.
// It can work with both filesystem.File and regular files from a path.
func getChecksum(fileOrPath interface{}) (string, error) {
	var file io.Reader

	// Determine if we're dealing with a filesystem.File or a regular file from a path
	switch v := fileOrPath.(type) {
	case filesystem.File:
		// If it's a filesystem.File, use it directly
		file = v
	case string:
		// If it's a string (a file path), open the file
		f, err := os.Open(v)
		if err != nil {
			return "", fmt.Errorf("failed to open file from path: %v", err)
		}
		defer f.Close()
		file = f
	default:
		return "", fmt.Errorf("invalid type, must be filesystem.File or string (file path)")
	}
	// Create a new SHA256 hash object
	hash := sha256.New() // hash is just an object which cannot be read or transformed by normal means. You need to use the hash.Sum(nil)-func!!!

	// Copy the file's content into the hash object
	_, err := io.Copy(hash, file)
	if err != nil {
		return "", fmt.Errorf("failed to compute checksum: %v", err)
	}

	// Compute the hash and return it as a hexadecimal string
	checksum := hex.EncodeToString(hash.Sum(nil)) // nil means raw hash without any prefix. A prefix is normally not needed.
	return checksum, nil
}

// Removes a file from OS or from a FSFileWithPath struct
// The interface type must be a FSFileWithPath or a string
func tidyUp(osFileOrFSFile interface{}) error {

	// var file io.Reader

	// Determine if we're dealing with a filesystem.File or a regular file from a path
	switch v := osFileOrFSFile.(type) {
	case FSFileWithPath:
		fs := v.FileSystem
		filePath := v.FilePath
		return fs.Remove(filePath)
	case string:
		return os.Remove(v)
	default:
		return fmt.Errorf("invalid type, must be FSFileWithPath or string (file path)")
	}
}
