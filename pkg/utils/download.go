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
	if _, err := os.Stat(dest); err == nil {
		return nil
	}

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

func DownloadWithCache(ctx context.Context, cacheDir, src, dest string, mode fs.FileMode) error {
	if _, err := os.Stat(dest); err == nil {
		return nil
	}

	cache, err := getCache(ctx, cacheDir, src, mode)
	if err != nil {
		return err
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

func getCache(ctx context.Context, cacheDir, src string, mode fs.FileMode) (string, error) {
	cache := filepath.Join(cacheDir, src)
	if _, err := os.Stat(cache); err == nil {
		return cache, nil
	}

	u, err := url.Parse(src)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "http", "https":
		cli := &http.Client{}
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			return "", err
		}
		req = req.WithContext(ctx)
		resp, err := cli.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return "", fmt.Errorf("%s: %s", u.String(), resp.Status)
		}

		err = os.MkdirAll(filepath.Dir(cache), 0755)
		if err != nil {
			return "", err
		}

		d, err := os.OpenFile(cache+".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			return "", err
		}

		_, err = io.Copy(d, resp.Body)
		if err != nil {
			d.Close()
			return "", err
		}
		d.Close()

		err = os.Rename(cache+".tmp", cache)
		if err != nil {
			return "", err
		}
		return cache, nil
	case "file", "":
		return u.Path, nil
	default:
		return "", fmt.Errorf("unknown scheme %v", u.Scheme)
	}
}
