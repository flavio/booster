package gzip

import (
	"compress/gzip"
	"github.com/pkg/errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Suffix is the name appended to files decompressed by this module
const Suffix = "_UNGZIPPED_BY_BOOSTER"

// DecompressAllIn uncompresses "recompressible" gzip files found in basePath and subdirectories
func DecompressAllIn(basePath string) error {
	return filepath.WalkDir(basePath, func(p string, d fs.DirEntry, err error) error {
		// skip irregular files
		if !d.Type().IsRegular() {
			return nil
		}
		// skip already decompressed (by name)
		if strings.HasSuffix(p, Suffix) {
			return nil
		}
		// skip already decompressed (by decompressed file existence)
		uncompressedPath := p + Suffix
		if _, err := os.Stat(uncompressedPath); err == nil {
			return nil
		}

		return decompress(p, uncompressedPath)
	})
}

// decompress decompresses a gzip file, if recompressible, into destinationPath
func decompress(sourcePath string, destinationPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return errors.Wrapf(err, "could not open to attempt decompression: %v", sourcePath)
	}

	rreader, err := NewRecompressibilityReader(source)
	if err == gzip.ErrHeader || err == io.EOF {
		// not a gzip or even empty, situation normal
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "error while initing decompression: %v", sourcePath)
	}

	destination, err := os.Create(destinationPath)
	if err != nil {
		return errors.Wrapf(err, "could not create temporary file to attempt decompression: %v", destinationPath)
	}

	_, err = io.Copy(destination, rreader)
	if err != nil {
		return errors.Wrapf(err, "error while decompressing: %v", sourcePath)
	}

	err = rreader.Close()
	if err != nil {
		return errors.Wrapf(err, "error while closing: %v", sourcePath)
	}
	err = destination.Close()
	if err != nil {
		return errors.Wrapf(err, "error while closing: %v", destinationPath)
	}
	err = source.Close()
	if err != nil {
		return errors.Wrapf(err, "error while closing: %v", sourcePath)
	}

	if !rreader.TransparentlyRecompressible() {
		// decompression worked but the result can't be compressed back
		// this archive can't be trusted, roll back
		os.Remove(destinationPath)
	}

	return nil
}

// RecompressAllIn recompresses any gzip files decompressed by DecompressAllIn
func RecompressAllIn(basePath string) error {
	return filepath.WalkDir(basePath, func(p string, d fs.DirEntry, err error) error {
		// skip any file other than those created by DecompressAllIn
		if !strings.HasSuffix(p, Suffix) {
			return nil
		}
		// skip already compressed
		compressedPath := strings.TrimSuffix(p, Suffix)
		if _, err := os.Stat(compressedPath); err == nil {
			return nil
		}

		err = compress(p, compressedPath)
		if err != nil {
			return errors.Wrapf(err, "decompression attempt failed: %v", compressedPath)
		}

		return nil
	})
}

// compress gzip-compresses a file
func compress(sourcePath string, destinationPath string) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return errors.Wrapf(err, "could not open to compress: %v", sourcePath)
	}

	destination, err := os.Create(destinationPath)
	if err != nil {
		return errors.Wrapf(err, "could not open to compress: %v", destinationPath)
	}

	gzDestination := gzip.NewWriter(destination)

	_, err = io.Copy(gzDestination, source)
	if err != nil {
		return errors.Wrapf(err, "error while compressing: %v", sourcePath)
	}

	err = gzDestination.Close()
	if err != nil {
		return errors.Wrapf(err, "error while closing: %v", destinationPath)
	}
	err = destination.Close()
	if err != nil {
		return errors.Wrapf(err, "error while closing: %v", destinationPath)
	}
	err = source.Close()
	if err != nil {
		return errors.Wrapf(err, "error while closing: %v", sourcePath)
	}

	return nil
}

// ListDecompressedOnly returns a set of all files in a directory
func ListDecompressedOnly(path string) map[string]bool {
	current := map[string]bool{}
	toRemove := make([]string, 0)
	filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		relative, _ := filepath.Rel(path, p)
		current[relative] = true

		if strings.HasSuffix(relative, Suffix) {
			toRemove = append(toRemove, strings.TrimSuffix(relative, Suffix))
		}
		return nil
	})

	// remove files for which we have an uncompressed copy
	for _, k := range toRemove {
		delete(current, k)
	}

	return current
}
