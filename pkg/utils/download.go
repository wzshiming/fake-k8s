package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func Download(ctx context.Context, src, dist string) error {
	file, err := openSrc(src)
	if err != nil {
		return err
	}
	defer file.Close()

	d, err := os.OpenFile(dist, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer d.Close()

	_, err = io.Copy(d, file)

	return err
}

func openSrc(file string) (io.ReadCloser, error) {
	u, err := url.Parse(file)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http", "https":
		cli := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
		resp, err := cli.Get(u.String())
		if err != nil {
			return nil, err
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
