package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"rsc.io/qr"
)

func getSSID() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		airportDir := "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources"
		if _, err := os.Stat(filepath.Join(airportDir, "airport")); os.IsNotExist(err) {
			return "", fmt.Errorf("airport not found at " + airportDir)
		}

		airportOut, err := exec.Command(filepath.Join(airportDir, "airport"), "-I").Output()
		if err != nil {
			return "", fmt.Errorf("command failed with error: " + err.Error())
		}
		var ssid string
		reader := bufio.NewReader(bytes.NewBuffer(airportOut))
		for {
			line, _, err := reader.ReadLine()
			if err == io.EOF {
				return "", fmt.Errorf("SSID not found")
			}
			if strings.Contains(string(line), "SSID") && !strings.Contains(string(line), "BSSID") {
				ssid = strings.Split(string(line), ": ")[1]
				break
			}
		}
		return ssid, nil
	case "windows":
		cmdOut, err := exec.Command("netsh", "wlan", "show", "interfaces").Output()
		if err != nil {
			return "", fmt.Errorf("command failed with error: " + err.Error())
		}
		var ssid string
		reader := bufio.NewReader(bytes.NewBuffer(cmdOut))
		for {
			line, _, err := reader.ReadLine()
			if err == io.EOF {
				return "", fmt.Errorf("SSID not found")
			}
			if strings.Contains(string(line), "SSID") && !strings.Contains(string(line), "BSSID") {
				ssid = strings.Split(string(line), ": ")[1]
				break
			}
		}
		return ssid, nil
	case "linux":
		return "", fmt.Errorf("linux not supported yet")
	default:
		return "", fmt.Errorf(runtime.GOOS + " not supported yet")
	}
}

func getPassword(ssid string) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		password, err := exec.Command("security", "find-generic-password", "-l", ssid, "-D", "AirPort network password", "-w").Output()
		if err != nil {
			return "", fmt.Errorf("failed to get password with error: " + err.Error())
		}
		return string(password), nil
	case "windows":
		cmdOut, err := exec.Command("netsh", "wlan", "show", "profile", "name=\""+ssid+"\"", "key=clear").Output()
		if err != nil {
			return "", fmt.Errorf("failed to get password with error: " + err.Error())
		}
		var password string
		reader := bufio.NewReader(bytes.NewBuffer(cmdOut))
		for {
			line, _, err := reader.ReadLine()
			if err == io.EOF {
				return "", fmt.Errorf("password not found")
			}
			if strings.Contains(string(line), "Key Content") {
				password = strings.Split(string(line), ": ")[1]
				break
			}
		}
		return password, nil
	default:
		return "", fmt.Errorf(runtime.GOOS + " not supported")
	}
}

func generateQRCode(ssid, password, path string) error {
	text := "WIFI:T:WPA;S:" + ssid + ";P:" + password + ";;"
	code, err := qr.Encode(text, qr.H)
	if err != nil {
		return err
	}
	imgByte := code.PNG()
	img, _, _ := image.Decode(bytes.NewReader(imgByte))

	qrPath := filepath.Join(path, ssid+"_password.png")

	out, err := os.Create(qrPath)
	if err != nil {
		return err
	}
	if err := png.Encode(out, img); err != nil {
		return err
	}
	fmt.Println("QR code saved to " + qrPath)
	return nil
}

func main() {
	makeQR := flag.Bool("qr", true, "Make a QR code")
	qrPath := flag.String("qr-path", ".", "Path to save generated QR code")

	flag.Parse()

	ssid, err := getSSID()
	if err != nil {
		panic(err)
	}

	password, err := getPassword(ssid)
	if err != nil {
		panic(err)
	}

	if *makeQR {
		if err := generateQRCode(ssid, password, *qrPath); err != nil {
			panic(err)
		}
	}

	fmt.Print(string(password))
}
