package utils

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

// CheckIfExistsAndIsDirectory checks is a path points at a directory.
func CheckIfExistsAndIsDirectory(path string) (fs.FileInfo, error) {
	stat, statErr := os.Stat(path)
	if statErr != nil {
		return nil, statErr // don't wrap OS errors:
	}
	if !stat.Mode().IsDir() {
		return nil, fmt.Errorf("not a directory: '%s'", path)
	}
	return stat, nil
}

// CheckIfExistsAndIsRegular checks is a path points at a regular file.
func CheckIfExistsAndIsRegular(path string) (fs.FileInfo, error) {
	stat, statErr := os.Stat(path)
	if statErr != nil {
		return nil, statErr // don't wrap OS errors:
	}
	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: '%s'", path)
	}
	return stat, nil
}

// CopyFile copies a file at the source path to the dest path.
func CopyFile(source, dest string, bufferSize int) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return nil
	}
	defer sourceFile.Close()
	destFile, err := os.Create(dest)
	if err != nil {
		return nil
	}
	defer destFile.Close()
	buf := make([]byte, bufferSize)
	for {
		n, err := sourceFile.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		if _, err := destFile.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}

// CreateRootFSFile uses dd to create a rootfs file of given size at a given path.
func CreateRootFSFile(path string, size int) error {
	exitCode, cmdErr := RunShellCommandNoSudo(fmt.Sprintf("dd if=/dev/zero of=%s bs=1M count=%d", path, size))
	if cmdErr != nil {
		return cmdErr
	}
	if exitCode != 0 {
		return fmt.Errorf("command finished with non-zero exit code")
	}
	return nil
}

// GetenvOrDefault calls os>lookup for a key and returns a fallback only if variable wasn't set.
func GetenvOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// MkfsExt4 uses mkfs.ext4 to create an EXT4 file system in a given file.
func MkfsExt4(path string) error {
	exitCode, cmdErr := RunShellCommandNoSudo(fmt.Sprintf("mkfs.ext4 %s", path))
	if cmdErr != nil {
		return cmdErr
	}
	if exitCode != 0 {
		return fmt.Errorf("command finished with non-zero exit code")
	}
	return nil
}

// Mount sudo mounts a rootfs file at a location.
func Mount(file, dir string) error {
	exitCode, cmdErr := RunShellCommandSudo(fmt.Sprintf("mount %s %s", file, dir))
	if cmdErr != nil {
		return cmdErr
	}
	if exitCode != 0 {
		return fmt.Errorf("command finished with non-zero exit code")
	}
	return nil
}

// MoveFile moves file from source to destination.
// os.Rename does not allow moving between drives
// hence we have to rewrite the file.
// Intermediate target directories will be created.
func MoveFile(source, target string) error {

	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}

	inputFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("couldn't open source file: %s", err)
	}
	outputFile, err := os.Create(target)
	if err != nil {
		inputFile.Close()
		return fmt.Errorf("couldn't open dest file: %s", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	inputFile.Close()
	if err != nil {
		return fmt.Errorf("writing to output file failed: %s", err)
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(source)
	if err != nil {
		return fmt.Errorf("failed removing original file: %s", err)
	}
	return nil
}

// PathExists returns true if path exists.
func PathExists(path string) (bool, error) {
	_, statErr := os.Stat(path)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return false, fmt.Errorf("path doesn't exists")
		}
		return false, statErr
	}
	// something exists:
	return true, nil
}

// RunShellCommandNoSudo runs a shell command without sudo.
func RunShellCommandNoSudo(command string) (int, error) {
	return runShellCommand(command, false)
}

// RunShellCommandSudo runs a shell command with sudo.
func RunShellCommandSudo(command string) (int, error) {
	return runShellCommand(command, true)
}

// Umount sudo umounts a location.
func Umount(dir string) error {
	exitCode, cmdErr := RunShellCommandSudo(fmt.Sprintf("umount %s", dir))
	if cmdErr != nil {
		return cmdErr
	}
	if exitCode != 0 {
		return fmt.Errorf("command finished with non-zero exit code")
	}
	return nil
}

// --

func runShellCommand(command string, sudo bool) (int, error) {
	if sudo {
		command = fmt.Sprintf("sudo %s", command)
	}
	cmd := exec.Command("/bin/sh", []string{`-c`, command}...)
	cmd.Stderr = os.Stderr
	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		return 1, fmt.Errorf("failed redirecting stdout: %+v", err)
	}
	if err := cmd.Start(); err != nil {
		return 1, fmt.Errorf("failed command start: %+v", err)
	}
	_, readErr := io.ReadAll(io.Reader(stdOut))
	if readErr != nil {
		return 1, fmt.Errorf("failed reading output: %+v", readErr)
	}
	if err := cmd.Wait(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode(), exitError
		}
		return 1, fmt.Errorf("failed waiting for command: %+v", err)
	}
	return 0, nil
}
