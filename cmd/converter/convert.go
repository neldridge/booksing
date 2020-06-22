package main

import (
	"os/exec"
	"strings"

	"github.com/gnur/booksing"
	log "github.com/sirupsen/logrus"
)

func convertBook(req booksing.ConvertRequest) error {

	l := log.WithField("filename", req.Filename)

	l.WithFields(log.Fields{
		"geturl": req.GetURL,
		"puturl": req.PutURL,
		"format": req.TargetFormat,
	}).Info("starting conversion")

	l.Info("downloading file")
	err := downloadFile(req.GetURL, req.Filename)
	if err != nil {
		log.WithField("err", err).Error("download failed")
		return err
	}

	newPath := strings.Replace(req.Filename, ".epub", "."+req.TargetFormat, 1)
	l.WithField("newpath", newPath).Info("converting file")
	cmd := exec.Command("ebook-convert", req.Filename, newPath)

	_, err = cmd.CombinedOutput()
	if err != nil {
		log.WithField("err", err).Error("conversion failed")
		return err
	}

	l.Info("uploading file")
	err = uploadFile(req.PutURL, newPath)
	if err != nil {
		log.WithField("err", err).Error("upload failed")
		return err
	}

	//TODO: let booksing know it is converted now
	return nil
}
