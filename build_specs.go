package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
)

func main() {
	cmd := exec.Command("smithy", "build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
	files, err := os.ReadDir("openapi")
	p := "openapi"
	if err != nil {
		fmt.Println("Error while reading openapi dir %v", err)
		os.Exit(1)
	}
	for _, entry := range files {
		if entry.IsDir() {
			pd, err := clearDirectory(path.Join(p, entry.Name()), func(e os.DirEntry, p string, d int) bool {
				return e.Name() == "openapi" && e.IsDir()
			}, 0)
			if err != nil {
				log.Fatal(err)
			}
			if pd != 1.0 {
				continue
			}
		}
		err = os.Remove(path.Join(p, entry.Name()))
		if err != nil {
			log.Fatal(err)
		}
	}
}

func clearDirectory(p string, validator func(os.DirEntry, string, int) bool, depth int) (float32, error) {
	entries, err := os.ReadDir(p)
	deleted := 0.0
	total := 0
	percentageDeleted := float32(0.0)
	if err != nil {
		return percentageDeleted, err
	}
	total = len(entries)
	for _, entry := range entries {
		valid := validator(entry, path.Join(p, entry.Name()), depth)
		if valid {
			continue
		}
		if entry.IsDir() {
			pd, err := clearDirectory(path.Join(p, entry.Name()), validator, depth+1)
			if err != nil {
				return percentageDeleted, err
			}
			if pd != 1.0 {
				continue
			}
		}
		err = os.Remove(path.Join(p, entry.Name()))
		if err != nil {
			return percentageDeleted, err
		}
		deleted++
	}
	percentageDeleted = float32(deleted / float64(total))
	return percentageDeleted, nil
}
