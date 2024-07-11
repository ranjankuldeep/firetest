package main

import (
	"fmt"
	"log"
	"os"
	"ranjankuldeep/test/snapshot"
)

func main() {
	baseFile := "../ubuntu-22.04.ext4"
	overlayDir := "../overlays"
	uid := 123
	gid := 100

	device, err := snapshot.CreateDeviceMapper(baseFile, overlayDir)
	if err != nil {
		log.Fatalf("Failed to create device mapper: %v", err)
	}
	defer func() {
		if err := device.Cleanup(); err != nil {
			log.Fatalf("Failed to cleanup device mapper: %v", err)
		}
	}()

	if err := os.Chown(device.OverlayFilename, uid, gid); err != nil {
		log.Fatalf("Failed to change ownership of overlay file: %v", err)
	}

	fmt.Printf("Ownership of overlay file %s changed to UID: %d and GID: %d\n", device.OverlayFilename, uid, gid)

	fmt.Printf("Device Mapper created:\n")
	fmt.Printf("Base Device: %s\n", device.BaseDev.Path())
	fmt.Printf("Overlay Device: %s\n", device.OverlayDev.Path())
	fmt.Printf("Base Name: %s\n", device.BaseName)
	fmt.Printf("Overlay Name: %s\n", device.OverlayName)
	fmt.Printf("Overlay Filename: %s\n", device.OverlayFilename)

	// The overlay device you will pass to Firecracker
	overlayDevicePath := fmt.Sprintf("/dev/mapper/%s", device.OverlayName)
	fmt.Printf("Overlay Device Path: %s\n", overlayDevicePath)

	// methods.ExampleJailerConfig_enablingJailer()
	select {}
}
