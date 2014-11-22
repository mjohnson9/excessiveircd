// Copyright (c) 2014 Michael Johnson. All rights reserved.
//
// Use of this source code is governed by the BSD license that can be found in
// the LICENSE file.

package config

import (
	"bufio"
	"encoding/gob"
	"errors"
	"flag"
	"os"
	"path/filepath"
)

var configDir = flag.String("config.directory", "./data", "Directory to use for configuration values")

// ErrDoesNotExist is an error that's returned when Get is called on a key that
// doesn't exist.
var ErrDoesNotExist = errors.New("does not exist")

// getConfigDir returns the configuration directory with a trailing slash.
func getConfigDir() string {
	dir, err := filepath.Abs(*configDir)
	if err != nil {
		panic(err)
	}

	return dir + "/"
}

func getConfigFileName(key string) (string, error) {
	configFile, err := filepath.Abs(getConfigDir() + key + ".gob")
	if err != nil {
		return "", err
	}
	return configFile, nil
}

// Set sets key to value. It uses encoding/gob to achieve this. Keys may contain
// / to specify directory. Note that this blocks until the value is successfuly
// written to disk or fails.
func Set(key string, value interface{}) error {
	configFile, err := getConfigFileName(key)
	if err != nil {
		return err
	}

	dir := filepath.Dir(configFile)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}

	f, err := os.OpenFile(configFile, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := bufio.NewWriter(f)
	enc := gob.NewEncoder(buf)

	if err := enc.Encode(value); err != nil {
		return err
	}

	if err := buf.Flush(); err != nil {
		return err
	}

	return nil
}

// Get gets a previously set key and puts it into val. It uses encoding/gob to
// achieve this, so val must be a pointer to the value.
// If key doesn't exist, ErrDoesNotExist is returned.
func Get(key string, val interface{}) error {
	configFile, err := getConfigFileName(key)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(configFile, os.O_RDONLY, 0666)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrDoesNotExist
		}
		return err
	}
	defer f.Close()

	dec := gob.NewDecoder(f)

	if err := dec.Decode(val); err != nil {
		return err
	}

	return nil
}
