package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

type Alias struct {
	Name    string
	Command string
}

type Config struct {
	GitHubToken string `json:"github_token"`
	GistID      string `json:"gist_id"`
}

type AliasManager struct {
	aliases       []Alias
	list          *widget.List
	window        fyne.Window
	selectedIndex int
	config        Config
}

// Version is set at build time via -ldflags "-X main.Version=..."
var Version = "dev"

//go:embed assets/icon.svg
var iconSVG []byte

func (am *AliasManager) loadAliases() error {
	home := os.Getenv("SNAP_REAL_HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "Loading aliases from: %s/.bash_aliases\n", home)
	file, err := os.Open(home + "/.bash_aliases")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "File does not exist, creating empty alias list\n")
			am.aliases = []Alias{}
			return nil
		}
		// Permission denied indicates confinement (snap) preventing dotfile access
		if os.IsPermission(err) {
			return fmt.Errorf("permission-denied")
		}
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		return err
	}
	defer file.Close()

	am.aliases = []Alias{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "alias ") && strings.Contains(line, "=") {
			parts := strings.SplitN(line[6:], "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				cmd := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
				am.aliases = append(am.aliases, Alias{Name: name, Command: cmd})
				fmt.Fprintf(os.Stderr, "Loaded alias: %s = %s\n", name, cmd)
			}
		}
	}
	fmt.Fprintf(os.Stderr, "Total aliases loaded: %d\n", len(am.aliases))
	return scanner.Err()
}

// importAliasesFromBytes loads aliases from the provided bytes
func (am *AliasManager) importAliasesFromBytes(content []byte) error {
	am.aliases = []Alias{}
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "alias ") && strings.Contains(line, "=") {
			parts := strings.SplitN(line[6:], "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				cmd := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
				am.aliases = append(am.aliases, Alias{Name: name, Command: cmd})
			}
		}
	}
	return scanner.Err()
}

// promptForAliasFile opens a file dialog to let user select an aliases file for import
func (am *AliasManager) promptForAliasFile() {
	fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()
		content, rerr := ioutil.ReadAll(reader)
		if rerr != nil {
			dialog.ShowError(rerr, am.window)
			return
		}
		if err := am.importAliasesFromBytes(content); err != nil {
			dialog.ShowError(err, am.window)
			return
		}
		am.refreshList()
	}, am.window)
	fd.SetFilter(storage.NewExtensionFileFilter([]string{"aliases", "txt", "sh"}))
	fd.Show()
}

func (am *AliasManager) saveAliases() error {
	home := os.Getenv("SNAP_REAL_HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	}
	file, err := os.Create(home + "/.bash_aliases")
	if err != nil {
		return err
	}
	defer file.Close()

	for _, alias := range am.aliases {
		fmt.Fprintf(file, "alias %s='%s'\n", alias.Name, alias.Command)
	}
	return nil
}

func (am *AliasManager) ensureBashrcSources() error {
	home := os.Getenv("SNAP_REAL_HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	}
	bashrcPath := home + "/.bashrc"
	file, err := os.Open(bashrcPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), ".bash_aliases") {
			return nil // already sourced
		}
	}

	// append to .bashrc
	f, err := os.OpenFile(bashrcPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("\n# Source bash aliases\nif [ -f ~/.bash_aliases ]; then\n    . ~/.bash_aliases\nfi\n")
	return err
}

func (am *AliasManager) loadConfig() error {
	home := os.Getenv("SNAP_REAL_HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	}
	configPath := home + "/.bash_alias_manager.json"
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			am.config = Config{}
			return nil
		}
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(&am.config)
}

func (am *AliasManager) saveConfig() error {
	home := os.Getenv("SNAP_REAL_HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	}
	configPath := home + "/.bash_alias_manager.json"
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(am.config)
}

func (am *AliasManager) refreshList() {
	am.list.Refresh()
}

// showAbout displays an about dialog with version and developer information
func (am *AliasManager) showAbout() {
	info := fmt.Sprintf("Bash Alias Manager\nVersion: %s\nDeveloper: ahrasel\nRepository: https://github.com/ahrasel/go-bash-alias-manager", Version)
	dialog.ShowInformation("About", info, am.window)
}

// checkForUpdate contacts GitHub Releases, checks latest tag and if newer offers to download and replace the binary.
func (am *AliasManager) checkForUpdate() {
	url := "https://api.github.com/repos/ahrasel/go-bash-alias-manager/releases/latest"
	resp, err := http.Get(url)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to check for update: %v", err), am.window)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		dialog.ShowError(fmt.Errorf("Update check failed: status %d", resp.StatusCode), am.window)
		return
	}
	var rel map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&rel); err != nil {
		dialog.ShowError(fmt.Errorf("Invalid response checking updates: %v", err), am.window)
		return
	}
	tag, _ := rel["tag_name"].(string)
	if tag == "" {
		dialog.ShowInformation("Update", "No release tag found", am.window)
		return
	}
	if !versionGreater(tag, Version) {
		dialog.ShowInformation("Update", "You are already running the latest version", am.window)
		return
	}
	confirm := dialog.NewConfirm("Update available", fmt.Sprintf("A new version %s is available (current %s). Update now?", tag, Version), func(ok bool) {
		if !ok {
			return
		}
		// find matching asset
		assets, _ := rel["assets"].([]interface{})
		goos := runtime.GOOS
		goarch := runtime.GOARCH
		// map GOARCH names to our asset naming (amd64 -> amd64, arm64 -> arm64)
		var assetURL string
		for _, a := range assets {
			m := a.(map[string]interface{})
			name := m["name"].(string)
			if strings.Contains(name, fmt.Sprintf("_%s_%s", goos, goarch)) {
				if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip") {
					assetURL = m["browser_download_url"].(string)
					break
				}
			}
		}
		if assetURL == "" {
			dialog.ShowError(fmt.Errorf("No matching release asset found for %s/%s", goos, goarch), am.window)
			return
		}
		// Download asset
		tmpd, err := os.MkdirTemp("", "bam-update-")
		if err != nil {
			dialog.ShowError(err, am.window)
			return
		}
		defer os.RemoveAll(tmpd)
		assetPath := filepath.Join(tmpd, "asset")
		out, err := os.Create(assetPath)
		if err != nil {
			dialog.ShowError(err, am.window)
			return
		}
		resp2, err := http.Get(assetURL)
		if err != nil {
			dialog.ShowError(err, am.window)
			out.Close()
			return
		}
		_, err = io.Copy(out, resp2.Body)
		resp2.Body.Close()
		out.Close()
		if err != nil {
			dialog.ShowError(err, am.window)
			return
		}
		// Extract and locate binary
		extractDir := filepath.Join(tmpd, "extracted")
		os.MkdirAll(extractDir, 0755)
		if strings.HasSuffix(assetPath, ".zip") {
			dialog.ShowInformation("Update", "Zip-based assets not yet supported for in-place update; please re-run the installer from the release page.", am.window)
			return
		} else {
			// tar.gz
			if err := exec.Command("tar", "-xzf", assetPath, "-C", extractDir).Run(); err != nil {
				dialog.ShowError(fmt.Errorf("Failed to extract update: %v", err), am.window)
				return
			}
		}
		// find binary
		var newBin string
		filepath.WalkDir(extractDir, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() && strings.Contains(d.Name(), "bash-alias-manager") && (d.Type()&0111 != 0) {
				newBin = p
				return io.EOF
			}
			return nil
		})
		if newBin == "" {
			dialog.ShowError(fmt.Errorf("Could not find new binary in archive"), am.window)
			return
		}
		// compute checksum of downloaded file for info
		data, _ := os.ReadFile(newBin)
		sum := sha256.Sum256(data)
		// attempt to replace current executable
		exe, err := os.Executable()
		if err != nil {
			dialog.ShowError(fmt.Errorf("Could not determine executable path: %v", err), am.window)
			return
		}
		// try to overwrite
		if err := os.WriteFile(exe, data, 0755); err != nil {
			// likely permission denied
			dialog.ShowError(fmt.Errorf("Failed to update in-place: %v. Please run the installer as described in the README.", err), am.window)
			return
		}
		dialog.ShowInformation("Update", fmt.Sprintf("Updated to %s (sha256: %s). Please restart the application.", tag, hex.EncodeToString(sum[:])), am.window)
	}, am.window)
	confirm.Show()
}

// versionGreater compares semantic versions like v1.2.3
func versionGreater(a, b string) bool {
	// strip leading v
	if strings.HasPrefix(a, "v") {
		a = a[1:]
	}
	if strings.HasPrefix(b, "v") {
		b = b[1:]
	}
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		ai := 0
		bi := 0
		if i < len(ap) {
			ai, _ = strconv.Atoi(ap[i])
		}
		if i < len(bp) {
			bi, _ = strconv.Atoi(bp[i])
		}
		if ai > bi {
			return true
		}
		if ai < bi {
			return false
		}
	}
	return false
}

func (am *AliasManager) backupToGist() {
	if am.config.GitHubToken == "" {
		tokenEntry := widget.NewPasswordEntry()
		tokenEntry.SetPlaceHolder("GitHub Personal Access Token")

		var d *dialog.CustomDialog
		form := &widget.Form{
			Items: []*widget.FormItem{
				{Text: "Token:", Widget: tokenEntry},
			},
			OnSubmit: func() {
				am.config.GitHubToken = tokenEntry.Text
				d.Hide()
				am.doBackup()
			},
		}
		d = dialog.NewCustom("Enter GitHub Token", "Cancel", form, am.window)
		d.Resize(fyne.NewSize(400, 100))
		d.Show()
		return
	}
	am.doBackup()
}

func (am *AliasManager) doBackup() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: am.config.GitHubToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Validate token
	_, _, err := client.Users.Get(ctx, "")
	if err != nil {
		dialog.ShowError(fmt.Errorf("Invalid GitHub token: %v", err), am.window)
		return
	}

	home := os.Getenv("SNAP_REAL_HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	content, err := os.ReadFile(home + "/.bash_aliases")
	if err != nil {
		if os.IsNotExist(err) {
			content = []byte("")
			// proceed with empty content
		} else if os.IsPermission(err) {
			// Cannot read dotfile due to confinement: ask user to select file to backup
			fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, rerr error) {
				if rerr != nil || reader == nil {
					return
				}
				defer reader.Close()
				b, _ := ioutil.ReadAll(reader)
				am.createGistFromContent(b)
			}, am.window)
			fd.SetFilter(storage.NewExtensionFileFilter([]string{"aliases", "txt", "sh"}))
			fd.Show()
			return
		} else {
			dialog.ShowError(err, am.window)
			return
		}
	}
	am.createGistFromContent(content)
}

func (am *AliasManager) createGistFromContent(content []byte) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: am.config.GitHubToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	files := map[github.GistFilename]github.GistFile{
		"bash_aliases": {Content: github.String(string(content))},
	}

	gist := &github.Gist{
		Description: github.String("Bash Aliases Backup"),
		Public:      github.Bool(false),
		Files:       files,
	}

	if am.config.GistID == "" {
		// Create new
		g, resp, err := client.Gists.Create(ctx, gist)
		if err != nil {
			// Provide clearer guidance for common permission errors (403/404) which often mean missing 'gist' scope
			if resp != nil && (resp.StatusCode == 403 || resp.StatusCode == 404) {
				dialog.ShowError(fmt.Errorf("Failed to create Gist (status %d). Ensure your GitHub token has the 'gist' scope and is valid. Error: %v", resp.StatusCode, err), am.window)
				return
			}
			dialog.ShowError(fmt.Errorf("Failed to create Gist: %v", err), am.window)
			return
		}
		am.config.GistID = *g.ID
		err = am.saveConfig()
		if err != nil {
			dialog.ShowError(err, am.window)
		}
	} else {
		// Update
		_, _, err := client.Gists.Edit(ctx, am.config.GistID, gist)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to update Gist: %v", err), am.window)
			return
		}
	}

	dialog.ShowInformation("Backup", "Aliases backed up to GitHub Gist successfully!", am.window)
}

func (am *AliasManager) restoreFromGist() {
	if am.config.GitHubToken == "" || am.config.GistID == "" {
		dialog.ShowInformation("Restore", "No backup found. Please backup first.", am.window)
		return
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: am.config.GitHubToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Validate token
	_, _, err := client.Users.Get(ctx, "")
	if err != nil {
		dialog.ShowError(fmt.Errorf("Invalid GitHub token: %v", err), am.window)
		return
	}

	gist, _, err := client.Gists.Get(ctx, am.config.GistID)
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to get Gist: %v", err), am.window)
		return
	}

	file, ok := gist.Files["bash_aliases"]
	if !ok {
		dialog.ShowError(fmt.Errorf("bash_aliases file not found in gist"), am.window)
		return
	}

	home := os.Getenv("SNAP_REAL_HOME")
	if home == "" {
		home, _ = os.UserHomeDir()
	}
	err = os.WriteFile(home+"/.bash_aliases", []byte(*file.Content), 0644)
	if err != nil {
		if os.IsPermission(err) {
			// Ask user to save file via portal
			fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, werr error) {
				if werr != nil || writer == nil {
					return
				}
				defer writer.Close()
				if _, werr := writer.Write([]byte(*file.Content)); werr != nil {
					dialog.ShowError(werr, am.window)
					return
				}
				// After saving, load the content into the app
				if lerr := am.importAliasesFromBytes([]byte(*file.Content)); lerr != nil {
					dialog.ShowError(lerr, am.window)
					return
				}
				am.refreshList()
			}, am.window)
			fd.SetFileName(".bash_aliases")
			fd.SetFilter(storage.NewExtensionFileFilter([]string{"aliases", "txt", "sh"}))
			fd.Show()
			return
		}
		dialog.ShowError(err, am.window)
		return
	}

	err = am.loadAliases()
	if err != nil {
		dialog.ShowError(err, am.window)
		return
	}
	am.refreshList()
	dialog.ShowInformation("Restore", "Aliases restored from GitHub Gist successfully!", am.window)
}

func (am *AliasManager) addAlias() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Alias name")
	cmdEntry := widget.NewEntry()
	cmdEntry.SetPlaceHolder("Command")

	var d *dialog.CustomDialog
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Name:", Widget: nameEntry},
			{Text: "Command:", Widget: cmdEntry},
		},
		OnSubmit: func() {
			if nameEntry.Text == "" || cmdEntry.Text == "" {
				return
			}
			am.aliases = append(am.aliases, Alias{Name: nameEntry.Text, Command: cmdEntry.Text})
			am.refreshList()
			err := am.saveAliases()
			if err != nil {
				dialog.ShowError(err, am.window)
			}
			d.Hide()
		},
	}

	d = dialog.NewCustom("Add Alias", "Cancel", form, am.window)
	d.Resize(fyne.NewSize(400, 200))
	d.Show()
}

func (am *AliasManager) editAlias(index int) {
	if index < 0 || index >= len(am.aliases) {
		return
	}
	alias := am.aliases[index]

	nameEntry := widget.NewEntry()
	nameEntry.SetText(alias.Name)
	cmdEntry := widget.NewEntry()
	cmdEntry.SetText(alias.Command)

	var d *dialog.CustomDialog
	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Name:", Widget: nameEntry},
			{Text: "Command:", Widget: cmdEntry},
		},
		OnSubmit: func() {
			if nameEntry.Text == "" || cmdEntry.Text == "" {
				return
			}
			am.aliases[index] = Alias{Name: nameEntry.Text, Command: cmdEntry.Text}
			am.refreshList()
			err := am.saveAliases()
			if err != nil {
				dialog.ShowError(err, am.window)
			}
			d.Hide()
		},
	}

	d = dialog.NewCustom("Edit Alias", "Cancel", form, am.window)
	d.Resize(fyne.NewSize(400, 200))
	d.Show()
}

func (am *AliasManager) deleteAlias(index int) {
	if index < 0 || index >= len(am.aliases) {
		return
	}
	confirm := dialog.NewConfirm("Delete Alias", "Are you sure you want to delete this alias?", func(confirmed bool) {
		if confirmed {
			am.aliases = append(am.aliases[:index], am.aliases[index+1:]...)
			am.refreshList()
			err := am.saveAliases()
			if err != nil {
				dialog.ShowError(err, am.window)
			}
		}
	}, am.window)
	confirm.Show()
}

func (am *AliasManager) reloadAliases() {
	err := am.loadAliases()
	if err != nil {
		dialog.ShowError(err, am.window)
		return
	}
	am.refreshList()
}

// saveAndReload removed: saving occurs immediately when aliases are added/edited/deleted

func main() {
	a := app.New()
	// Set embedded app icon when available
	if len(iconSVG) > 0 {
		res := fyne.NewStaticResource("icon.svg", iconSVG)
		a.SetIcon(res)
	}
	w := a.NewWindow("Bash Alias Manager")

	am := &AliasManager{window: w, selectedIndex: -1}
	err := am.loadAliases()
	if err != nil {
		if err.Error() == "permission-denied" {
			// Snap confined: ask user to import their aliases via file chooser
			resp := dialog.NewConfirm("Permission Denied", "Cannot access ~/.bash_aliases due to sandboxing. Would you like to select the file to import?", func(confirmed bool) {
				if confirmed {
					am.promptForAliasFile()
				}
			}, w)
			resp.Show()
		} else {
			dialog.ShowError(err, w)
		}
	}

	// Try to ensure .bashrc sources .bash_aliases, but if permission denied, instruct user
	if err := am.ensureBashrcSources(); err != nil {
		if os.IsPermission(err) {
			d := dialog.NewConfirm("Permission Denied", "Cannot edit ~/.bashrc due to sandboxing. To ensure aliases are loaded, please add the following lines to your ~/.bashrc manually:\n\nif [ -f ~/.bash_aliases ]; then\n    . ~/.bash_aliases\nfi\n\nWould you like to open the .bashrc file to edit it?", func(confirmed bool) {
				if confirmed {
					// Let user select the .bashrc file via portal
					fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
						if err != nil || reader == nil {
							return
						}
						defer reader.Close()
						// We open the file for the user to edit in their editor manually; we don't write it ourselves
					}, w)
					fd.SetFilter(storage.NewExtensionFileFilter([]string{"bashrc", "sh", "txt"}))
					fd.Show()
				}
			}, w)
			d.Show()
		}
	}

	err = am.loadConfig()
	if err != nil {
		dialog.ShowError(err, w)
	}

	am.list = widget.NewList(
		func() int {
			return len(am.aliases)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(fmt.Sprintf("%s = %s", am.aliases[i].Name, am.aliases[i].Command))
		},
	)

	am.list.OnSelected = func(id widget.ListItemID) {
		am.selectedIndex = int(id)
	}

	addBtn := widget.NewButton("Add", am.addAlias)
	editBtn := widget.NewButton("Edit", func() {
		if am.selectedIndex >= 0 && am.selectedIndex < len(am.aliases) {
			am.editAlias(am.selectedIndex)
		}
	})
	deleteBtn := widget.NewButton("Delete", func() {
		if am.selectedIndex >= 0 && am.selectedIndex < len(am.aliases) {
			am.deleteAlias(am.selectedIndex)
		}
	})
	// Save button removed (save happens automatically when editing/adding/removing aliases)
	reloadBtn := widget.NewButton("Reload", am.reloadAliases)
	backupBtn := widget.NewButton("Backup", am.backupToGist)
	restoreBtn := widget.NewButton("Restore", am.restoreFromGist)
	updateBtn := widget.NewButton("Update", am.checkForUpdate)
	aboutBtn := widget.NewButton("About", am.showAbout)

	buttonBox := container.NewHBox(addBtn, editBtn, deleteBtn, reloadBtn, backupBtn, restoreBtn, updateBtn, aboutBtn)

	w.SetContent(container.NewBorder(
		nil,
		buttonBox,
		nil,
		nil,
		am.list,
	))

	w.Resize(fyne.NewSize(600, 400))
	w.ShowAndRun()
}
