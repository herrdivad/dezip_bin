# **fat32_reader**

A Go program to extract files from a FAT32 `.bin` disk image (e.g., `piusb.bin`), optionally verifying checksums and cleaning up files afterward. Primarily tested with the [go-diskfs](https://github.com/diskfs/go-diskfs) library, it can open a partition from the image, list directories, and selectively extract files based on provided CLI arguments.

---

## **Table of Contents**

- [Project Overview](#project-overview)
- [Features](#features)
- [Usage](#usage)
  - [CLI Arguments](#cli-arguments)
  - [Examples](#examples)
- [Build Instructions](#build-instructions)
  - [Building for Raspberry Pi Zero W](#building-for-raspberry-pi-zero-w)
- [Functions Explained](#functions-explained)
  - [main()](#main)
  - [readPath()](#readpath)
  - [readAndExtractFile()](#readandextractfile)
  - [loopThroughFilesToExtract()](#loopthroughfilestoextract)
  - [getChecksum()](#getchecksum)
  - [tidyUp()](#tidyup)
- [License](#license)

---

## **Project Overview**

This script:
1. Opens a FAT32 `.bin` disk image (e.g., `piusb.bin`) using [go-diskfs](https://github.com/diskfs/go-diskfs).
2. Lists files and directories (and can recurse if desired).
3. Sorts the files by modification time, optionally verifies checksums, and extracts them to a local target directory.
4. Cleans up (removes) extracted files if desired.

**Primary Goal**: Automate extraction of the latest file(s) from a disk image, such as those created by certain instruments or devices.

---

## **Features**

- **Opens the first partition** of a `.bin` image and lists files/folders (FAT32).
- **Sorts** the files by modification time (descending).
- **Optional Checksum** comparison for duplicates.
- **Selective file extraction** based on arguments: `--latest`, `--all`, `--latestSame`.
- **Cleanup option** to remove files after successful extraction, either from:
  - The host OS (`os.Remove`)  
  - The FAT32 filesystem (`filesystem.FileSystem.Remove`)

---

## **Usage**

### **CLI Arguments**

1. **Argument 1**: `<device>` – If empty, works on the root directory (`/`).  
   - If `SMP50` or another name is provided, it will target `/<device>/results`.  
   - If `all`, it will recurse from `/`.  

2. **Argument 2**: `<rangeArgument>` – Usually `--latest`, `--latestSame`, or `--all`.  
   - `--latest`: Extract only the single newest file.  
   - `--latestSame`: Attempt to extract additional files with the same basename.  
   - `--all`: Extract all files in the sorted list.  

3. **Argument 3**: `<shouldTidyUp>` – A boolean (`true`/`false`). If `true`, files in the FAT32 image are removed after extraction, or the local file system path is removed (depending on usage).

### **Examples**

```bash
# 1) Running without arguments:
#    - Targets root directory
#    - Sorts by modification time
#    - Extracts the newest file only
./fat32_reader

# 2) Running with device SMP50 (no second argument -> default is "--latest"):
#    - Moves into /SMP50/results
#    - Extracts the newest file only
./fat32_reader SMP50

# 3) Extract all files in "/SMP50/results" and do not tidy up:
./fat32_reader SMP50 --all false

# 4) Extract only the newest file, 
#    then remove it from the .bin FS once done:
./fat32_reader SMP50 --latest true
```

---

## **Build Instructions**

### **General Build (for Linux)**

```bash
go mod tidy
go build -o fat32_reader
```

This produces a binary named `fat32_reader` (or `fat32_reader.exe` on Windows).

### **Building for Raspberry Pi Zero W**

A **Raspberry Pi Zero W** uses **ARMv6**. So you need to cross-compile with:

```bash
# For 32-bit ARMv6 (Pi Zero / Zero W):
GOOS=linux GOARCH=arm GOARM=6 go build -o fat32_reader
```

Then transfer `fat32_reader` to your Pi Zero W (e.g., via `scp`). Once on the Pi, you can run:

```bash
chmod +x fat32_reader
./fat32_reader
```

---

## **Functions Explained**

### **main()**
1. **Parses `os.Args`** for optional arguments:
   - `[1]`: Device (e.g., `SMP50`).
   - `[2]`: `--latest`, `--latestSame`, `--all` (default `--latest`).
   - `[3]`: Boolean for `shouldTidyUp`.
2. **Opens the disk image** with `diskfs.Open(imagePath)`.
3. **Reads the first partition** with `disk.GetFilesystem(0)`.
4. **Lists files** using `readPath()`, optionally recurses depending on arguments.
5. **Sorts files** by modification time, ignoring “System Volume Information”.
6. **Calls `loopThroughFilesToExtract()`** to handle extraction based on your rules.

### **readPath(myFS, mypath, recursive) -> []FileWithPath**
- Reads a path `mypath` in the `myFS` filesystem.
- Ignores special directories like `"."`, `".."`, `"System Volume Information"`.
- If `recursive == true`, it calls itself on subdirectories.
- Returns a slice of `FileWithPath`, which includes the `fs.FileInfo` and the `ItsPath`.

### **loopThroughFilesToExtract(files, myFS) -> bool**
- Iterates through sorted files.
- Checks arguments (`--latest`, `--latestSame`, `--all`) and calls `readAndExtractFile()`.
- If argument `[3]` is set to `true`, calls `tidyUp()` on the extracted file(s).
- Returns `true` if an error condition arises, otherwise `false`.

### **readAndExtractFile(file, myFS, check ...bool) -> bool**
1. **file**: A `FileWithPath` containing `fs.FileInfo` and a path.
2. **Opens the file** with `myFS.OpenFile(absolutePath, 0)`.
3. **Optionally calculates checksums** if `check[0] == true`.
4. **Creates the local output directory** (under `targetDir`).
5. **Copies** the data from the `filesystem.File` to an output file.
6. Returns `true` if successful, `false` if any error.

### **getChecksum(fileOrPath interface{}) -> (string, error)**
- Can handle either a `filesystem.File` or a local file path (`string`).
- Uses a **SHA-256** hash, reading with `io.Copy`.
- Returns the hex-encoded result via `hex.EncodeToString`.

### **tidyUp(osFileOrFSFile interface{}) -> error**
- Switches on type:
  - **FSFileWithPath**: Calls `fs.Remove(filePath)`.
  - **string**: Calls `os.Remove(path)`.
- Lets you remove a file from the disk image’s FS or from the local system, depending on what is passed.

---

## **License**

This project is licensed under the **MIT License**. See the header in `fat32_reader.go` for the full text:

```
MIT License

Copyright (c) 2024 David Herrmann
...
```

You’re free to use, modify, and distribute this software, provided the license text remains intact.

---

### **Feedback or Contributions**
Feel free to open issues or pull requests to improve or extend functionality.
This docu was initial created by ChatGPT AI.
