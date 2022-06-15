package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func Untar(src string, filter func(file string) (string, bool)) error {
	if strings.HasSuffix(src, ".tar.gz") {
		return Untargz(src, filter)
	} else if strings.HasSuffix(src, ".zip") {
		return Unzip(src, filter)
	}
	return fmt.Errorf("unsupported archive format: %s", src)
}

func Unzip(src string, filter func(file string) (string, bool)) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fi := f.FileInfo()

		name := f.Name
		if fi.IsDir() {
			continue
		}

		err = func() error {
			name, ok := filter(name)
			if !ok {
				return nil
			}

			err = os.MkdirAll(filepath.Dir(name), 0755)
			if err != nil {
				return err
			}

			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			outFile, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fi.Mode())
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, rc)
			if err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

func Untargz(src string, filter func(file string) (string, bool)) error {
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		name := hdr.Name
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		err = func() error {
			name, ok := filter(name)
			if !ok {
				return nil
			}

			err = os.MkdirAll(filepath.Dir(name), 0755)
			if err != nil {
				return err
			}
			outFile, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fs.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tr); err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}
