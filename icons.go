package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/png"
	"log"

	ico "github.com/sergeymakinen/go-ico"
)

//go:embed all:icons/*.ico
var icons embed.FS

var (
	trayIcons = map[string][]byte{}
)

type Icons []image.Image

func (ic *Icons) Get(size int) ([]byte, error) {
	for _, i := range *ic {
		if i.Bounds().Dx() == size {
			buf := new(bytes.Buffer)
			if err := png.Encode(buf, i); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
	}
	return nil, fmt.Errorf("not found size:%d", size)
}

func loadIcons(name string) (Icons, error) {
	b, err := icons.ReadFile(name)
	if err != nil {
		return nil, err
	}
	images, err := ico.DecodeAll(bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	return images, nil
}

func init() {
	light, err := loadIcons("icons/light.ico")
	if err != nil {
		log.Fatal(err)
	}
	light16, err := light.Get(16)
	if err != nil {
		log.Fatal(err)
	}
	trayIcons["light"] = light16
	dark, err := loadIcons("icons/dark.ico")
	if err != nil {
		log.Fatal(err)
	}
	dark16, err := dark.Get(16)
	if err != nil {
		log.Fatal(err)
	}
	trayIcons["dark"] = dark16
	record, err := loadIcons("icons/record.ico")
	if err != nil {
		log.Fatal(err)
	}
	record16, err := record.Get(16)
	if err != nil {
		log.Fatal(err)
	}
	trayIcons["record"] = record16
}
