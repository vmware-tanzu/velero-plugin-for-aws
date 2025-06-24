package main

import (
	"bytes"
	"io"

	"filippo.io/age"
)

type Age struct {
	recipient string
	key       string
}

func NewAge(recipient, key string) *Age {
	return &Age{recipient: recipient, key: key}
}

func (a *Age) Encrypt(src io.Reader) (io.ReadCloser, error) {
	recipient, err := age.ParseX25519Recipient(a.recipient)
	if err != nil {
		return nil, err
	}

	var outBuf bytes.Buffer

	w, err := age.Encrypt(&outBuf, recipient)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(w, src); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	return io.NopCloser(bytes.NewReader(outBuf.Bytes())), nil
}

func (a *Age) Decrypt(src io.Reader) (io.ReadCloser, error) {
	identity, err := age.ParseX25519Identity(a.key)
	if err != nil {
		return nil, err
	}

	plainR, err := age.Decrypt(src, identity)
	if err != nil {
		return nil, err
	}

	var outBuf bytes.Buffer
	if _, err := io.Copy(&outBuf, plainR); err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(outBuf.Bytes())), nil
}
