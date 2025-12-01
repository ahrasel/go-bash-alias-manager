package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v53/github"
	"golang.org/x/oauth2"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
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

func (am *AliasManager) loadAliases() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	file, err := os.Open(home + "/.bash_aliases")
	if err != nil {
		if os.IsNotExist(err) {
			am.aliases = []Alias{}
			return nil
		}
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
			}
		}
	}
	return scanner.Err()
}

func (am *AliasManager) saveAliases() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
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
	home, err := os.UserHomeDir()
	if err != nil {
		return err
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
	home, err := os.UserHomeDir()
	if err != nil {
		return err
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
	home, err := os.UserHomeDir()
	if err != nil {
		return err
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

	home, _ := os.UserHomeDir()
	content, err := os.ReadFile(home + "/.bash_aliases")
	if err != nil {
		if os.IsNotExist(err) {
			content = []byte("")
		} else {
			dialog.ShowError(err, am.window)
			return
		}
	}

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
		g, _, err := client.Gists.Create(ctx, gist)
		if err != nil {
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

	home, _ := os.UserHomeDir()
	err = os.WriteFile(home+"/.bash_aliases", []byte(*file.Content), 0644)
	if err != nil {
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

func (am *AliasManager) saveAndReload() {
	err := am.saveAliases()
	if err != nil {
		dialog.ShowError(err, am.window)
		return
	}
	// Reload shell or something? For now, just save
	exec.Command("bash", "-c", "source ~/.bashrc").Run() // optional
}

func main() {
	a := app.New()
	w := a.NewWindow("Bash Alias Manager")

	am := &AliasManager{window: w, selectedIndex: -1}
	err := am.loadAliases()
	if err != nil {
		dialog.ShowError(err, w)
	}

	am.ensureBashrcSources() // ensure .bashrc sources .bash_aliases

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
	saveBtn := widget.NewButton("Save", am.saveAndReload)
	reloadBtn := widget.NewButton("Reload", am.reloadAliases)
	backupBtn := widget.NewButton("Backup", am.backupToGist)
	restoreBtn := widget.NewButton("Restore", am.restoreFromGist)

	buttonBox := container.NewHBox(addBtn, editBtn, deleteBtn, saveBtn, reloadBtn, backupBtn, restoreBtn)

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