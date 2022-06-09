package utils

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

func DownloadWithCacheAndExtract(ctx context.Context, cacheDir, src, dest string, match string, mode fs.FileMode) error {
	cacheTar, err := getCache(ctx, cacheDir, src, 0644)
	if err != nil {
		return err
	}
	cache := filepath.Join(filepath.Dir(cacheTar), match)
	if _, err := os.Stat(cache); err != nil {
		err = Untar(cacheTar, func(file string) (string, bool) {
			if filepath.Base(file) == match {
				return cache, true
			}
			return "", false
		})
		if err != nil {
			return err
		}
		os.Chmod(cache, mode)
	}

	// link the cache file to the dest file
	err = os.Symlink(cache, dest)
	if err != nil {
		return err
	}
	return nil
}

func getCache(ctx context.Context, cacheDir, src string, mode fs.FileMode) (string, error) {
	cache := filepath.Join(cacheDir, src)
	if _, err := os.Stat(cache); err != nil {
		err := Download(ctx, src, cache+".tmp", mode)
		if err != nil {
			return "", err
		}
		err = os.Rename(cache+".tmp", cache)
		if err != nil {
			return "", err
		}
	}
	return cache, nil
}

func DownloadWithCache(ctx context.Context, cacheDir, src, dest string, mode fs.FileMode) error {
	cache, err := getCache(ctx, cacheDir, src, mode)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dest); err == nil {
		return nil
	}

	err = os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		return err
	}

	// link the cache file to the dest file
	err = os.Symlink(cache, dest)
	if err != nil {
		return err
	}
	return nil
}

func Download(ctx context.Context, src, dist string, mode fs.FileMode) error {
	err := os.MkdirAll(filepath.Dir(dist), 0755)
	if err != nil {
		return err
	}

	file, err := openSrc(ctx, src)
	if err != nil {
		return err
	}
	defer file.Close()

	d, err := os.OpenFile(dist, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer d.Close()

	_, err = io.Copy(d, file)

	return err
}

func openSrc(ctx context.Context, file string) (io.ReadCloser, error) {
	u, err := url.Parse(file)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http", "https":
		cli := &http.Client{}
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			return nil, err
		}
		req = req.WithContext(ctx)
		resp, err := cli.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("%s: %s", u.String(), resp.Status)
		}
		return resp.Body, nil
	case "file", "":
		file, err := os.OpenFile(u.Path, os.O_RDONLY, 0)
		if err != nil {
			return nil, err
		}
		return file, nil
	default:
		return nil, fmt.Errorf("unknown scheme %v", u.Scheme)
	}
}
