package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/peteretelej/nasa"
)

var (
	random     = flag.Bool("random", true, "use random pictures, if false will only display today's APOD")
	interval   = flag.Duration("interval", time.Minute*10, "interval to change wallpaper")
	cmdString  = flag.String("cmd", "", "command string to change the wallpaper")
	cmdDefault = flag.String("cmdDefault", "", "use a default command to set the wallpaper")
	cmds       []string

	cmdDefaults = map[string]string{
		"gnome":   "gsettings set org.gnome.desktop.background picture-uri file://%s",
		"kde":     "dcop kdesktop KBackgroundIface setWallpaper %s 1",
		"gnome2":  "gconftool-2 --set /desktop/gnome/background/picture_filename --type=string %s",
		"xfce":    "xfconf-query -c xfce4-desktop -p /backdrop/screen0/monitor0/image-path -s %s",
		"mate":    `dconf write /org/mate/desktop/background/picture-filename "%s"`,
		"pcmanfm": "pcmanfm -w %s --wallpaper-mode=fit", //Lubuntu
		"feh":     "feh --bg-scale %s",
		"setroot": "setroot %s",
	}
)

func init() {
	if os.Getenv("NASAKEY") == "" {
		fmt.Println(nasa.APIKEYMissing)
	}
}

func main() {
	flag.Parse()
	realCmdString := getCmdString()
	if realCmdString == "" {
		log.Fatal("wallpapers change command not found, set custom one with -cmd")
	}
	cmds = strings.Split(fmt.Sprintf(realCmdString, tmpfile), " ")
	if !*random {
		if err := todaysAPOD(); err != nil {
			log.Fatalf("nasa-wallpapers: %v\n", err)
		}
	}

	if err := randomAPOD(*interval); err != nil {
		log.Fatalf("nasa-wallpapers: %v\n", err)
	}
}

func todaysAPOD() error {
	defer cleanUp()
	return errors.New("TODO")
}

func randomAPOD(interval time.Duration) error {
	defer cleanUp()
	if interval < time.Second {
		return errors.New("interval set is too low")
	}

	fmt.Printf("nasa-wallpapers: resetting wallpaper to a random NASA APOD picture every %s\n", interval)
	for {
		fails := 0
		var err error
		for i := 0; i < 3; i++ {
			err = updateRandom()
			if err == nil {
				fails = 0
				break
			}
			fails++
		}
		if fails != 0 {
			log.Printf("nasa-wallpapers: unable to fetch wallpapers: %v", err)
		}
		time.Sleep(interval)
	}
}

func getCmdString() string {
	if *cmdString != "" {
		return *cmdString
	}
	if cmd, ok := cmdDefaults[*cmdDefault]; ok {
		return cmd
	}
	return ""
}

func updateRandom() error {
	apod, err := nasa.RandomAPOD()
	if err != nil {
		return err
	}
	if apod.HDURL == "" {
		return errors.New("invalid response from NASA API")
	}
	req, err := http.NewRequest("GET", apod.HDURL, nil)
	if err != nil {
		return err
	}
	cl := &http.Client{Timeout: time.Second * 40}

	resp, err := cl.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("NASA API Response not OK: %d %s",
			resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	defer func() { _ = resp.Body.Close() }()
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(dat) < 512 {
		return errors.New("invalid response from APOD image url")
	}
	switch http.DetectContentType(dat[:512]) {
	case "image/jpg", "image/jpeg", "image/png", "image/gif":
	default:
		return errors.New("Apod returned is not a valid image mimetype")
	}
	err = ioutil.WriteFile(tmpfile, dat, 0644)
	if err != nil {
		return err
	}

	_, err = exec.Command(cmds[0], cmds[1:]...).Output()
	return err
}

var tmpfile string

func init() {
	tmp, err := ioutil.TempFile("", "nasa-wallpapers-pic.jpg")
	if err != nil {
		log.Fatalf("unable to get tempfile to work with: %v", err)
	}
	tmpfile = tmp.Name()
	if err := tmp.Close(); err != nil {
		log.Fatalf("unable to use tempfile %v", err)
	}
}

func cleanUp() {
	if tmpfile == "" {
		return
	}
	if _, err := os.Stat(tmpfile); err != nil {
		return
	}
	if err := os.Remove(tmpfile); err != nil {
		log.Printf("unable to clean up: %v", err)
	}
}
