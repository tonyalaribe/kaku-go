package main

import (
	"bytes"
	"fmt"
	"image/jpeg"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"path"

	"code.google.com/p/go-uuid/uuid"
	"github.com/mitchellh/goamz/s3"
	"github.com/nfnt/resize"
)

func randSeq(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func (c *appContext) uploadPic(a *multipart.FileHeader) (string, string) {
	log.Println("In upload pic territory")

	bucket := c.bucket

	file, err := a.Open()
	defer file.Close()
	if err != nil {
		panic(err.Error())
	}

	if err != nil {
		panic(err)
	}

	buf, _ := ioutil.ReadAll(file)

	fn := uuid.New()
	fname := "places/" + fn + path.Ext(a.Filename)
	thumbname := "placesthumb/" + fn + path.Ext(a.Filename)
	log.Println(fname)

	b := "http://s3-us-west-2.amazonaws.com/" + c.bucket.Name + "/" + fname
	d := "http://s3-us-west-2.amazonaws.com/" + c.bucket.Name + "/" + thumbname

	filetype := http.DetectContentType(buf)

	err = bucket.Put(fname, buf, filetype, s3.PublicRead)

	if err != nil {
		log.Println("bucket put error for main image")
		panic(err.Error())

	}
	log.Print("added a full image")
	img, err := jpeg.Decode(bytes.NewReader(buf))

	if err != nil {
		log.Println(err.Error())
	}

	m := resize.Resize(200, 200, img, resize.Lanczos2)

	buf2 := new(bytes.Buffer)
	err = jpeg.Encode(buf2, m, nil)
	if err != nil {
		fmt.Println(err.Error())
	}
	thumb := buf2.Bytes()
	filetype2 := http.DetectContentType(thumb)
	err = bucket.Put(thumbname, thumb, filetype2, s3.PublicRead)

	if err != nil {
		log.Println("bucket put error for thumbnail")
		panic(err.Error())
	}
	log.Println("uploaded one thumb image")
	return b, d

}
