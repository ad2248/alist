package onedrive

import (
	"fmt"
	"github.com/Xhofe/alist/conf"
	"github.com/Xhofe/alist/drivers/base"
	"github.com/Xhofe/alist/model"
	"github.com/Xhofe/alist/utils"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"path/filepath"
)

type Onedrive struct{}


func (driver Onedrive) Config() base.DriverConfig {
	return base.DriverConfig{
		Name: "Onedrive",
		OnlyProxy: false,
	}
}

func (driver Onedrive) Items() []base.Item {
	return []base.Item{
		{
			Name:        "zone",
			Label:       "zone",
			Type:        base.TypeSelect,
			Required:    true,
			Values:      "global,cn,us,de",
			Description: "",
		},
		{
			Name:     "internal_type",
			Label:    "onedrive type",
			Type:     base.TypeSelect,
			Required: true,
			Values:   "onedrive,sharepoint",
		},
		{
			Name:     "client_id",
			Label:    "client id",
			Type:     base.TypeString,
			Required: true,
		},
		{
			Name:     "client_secret",
			Label:    "client secret",
			Type:     base.TypeString,
			Required: true,
		},
		{
			Name:     "redirect_uri",
			Label:    "redirect uri",
			Type:     base.TypeString,
			Required: true,
		},
		{
			Name:     "refresh_token",
			Label:    "refresh token",
			Type:     base.TypeString,
			Required: true,
		},
		{
			Name:     "site_id",
			Label:    "site id",
			Type:     base.TypeString,
			Required: false,
		},
		{
			Name:     "root_folder",
			Label:    "root folder path",
			Type:     base.TypeString,
			Required: false,
		},
		{
			Name:     "order_by",
			Label:    "order_by",
			Type:     base.TypeSelect,
			Values:   "name,size,lastModifiedDateTime",
			Required: false,
		},
		{
			Name:     "order_direction",
			Label:    "order_direction",
			Type:     base.TypeSelect,
			Values:   "asc,desc",
			Required: false,
		},
	}
}

func (driver Onedrive) Save(account *model.Account, old *model.Account) error {
	_, ok := onedriveHostMap[account.Zone]
	if !ok {
		return fmt.Errorf("no [%s] zone", account.Zone)
	}
	if old != nil {
		conf.Cron.Remove(cron.EntryID(old.CronId))
	}
	account.RootFolder = utils.ParsePath(account.RootFolder)
	err := driver.RefreshToken(account)
	if err != nil {
		return err
	}
	cronId, err := conf.Cron.AddFunc("@every 1h", func() {
		name := account.Name
		log.Debugf("onedrive account name: %s", name)
		newAccount, ok := model.GetAccount(name)
		log.Debugf("onedrive account: %+v", newAccount)
		if !ok {
			return
		}
		err = driver.RefreshToken(&newAccount)
		_ = model.SaveAccount(&newAccount)
	})
	if err != nil {
		return err
	}
	account.CronId = int(cronId)
	err = model.SaveAccount(account)
	if err != nil {
		return err
	}
	return nil
}

func (driver Onedrive) File(path string, account *model.Account) (*model.File, error) {
	path = utils.ParsePath(path)
	if path == "/" {
		return &model.File{
			Id:        account.RootFolder,
			Name:      account.Name,
			Size:      0,
			Type:      conf.FOLDER,
			Driver:    driver.Config().Name,
			UpdatedAt: account.UpdatedAt,
		}, nil
	}
	dir, name := filepath.Split(path)
	files, err := driver.Files(dir, account)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.Name == name {
			return &file, nil
		}
	}
	return nil, base.ErrPathNotFound
}

func (driver Onedrive) Files(path string, account *model.Account) ([]model.File, error) {
	path = utils.ParsePath(path)
	cache, err := base.GetCache(path, account)
	if err == nil {
		files, _ := cache.([]model.File)
		return files, nil
	}
	rawFiles, err := driver.GetFiles(account, path)
	if err != nil {
		return nil, err
	}
	files := make([]model.File, 0)
	for _, file := range rawFiles {
		files = append(files, *driver.FormatFile(&file))
	}
	if len(files) > 0 {
		_ = base.SetCache(path, files, account)
	}
	return files, nil
}

func (driver Onedrive) Link(path string, account *model.Account) (*base.Link, error) {
	file, err := driver.GetFile(account, path)
	if err != nil {
		return nil, err
	}
	if file.File.MimeType == "" {
		return nil, base.ErrNotFile
	}
	link := base.Link{
		Url: file.Url,
	}
	return &link, nil
}

func (driver Onedrive) Path(path string, account *model.Account) (*model.File, []model.File, error) {
	log.Debugf("onedrive path: %s", path)
	file, err := driver.File(path, account)
	if err != nil {
		return nil, nil, err
	}
	if !file.IsDir() {
		//file.Url, _ = driver.Link(path, account)
		return file, nil, nil
	}
	files, err := driver.Files(path, account)
	if err != nil {
		return nil, nil, err
	}
	return nil, files, nil
}

func (driver Onedrive) Proxy(c *gin.Context, account *model.Account) {
	c.Request.Header.Del("Origin")
}

func (driver Onedrive) Preview(path string, account *model.Account) (interface{}, error) {
	return nil, base.ErrNotSupport
}

func (driver Onedrive) MakeDir(path string, account *model.Account) error {
	return base.ErrNotImplement
}

func (driver Onedrive) Move(src string, dst string, account *model.Account) error {
	return base.ErrNotImplement
}

func (driver Onedrive) Copy(src string, dst string, account *model.Account) error {
	return base.ErrNotImplement
}

func (driver Onedrive) Delete(path string, account *model.Account) error {
	return base.ErrNotImplement
}

func (driver Onedrive) Upload(file *model.FileStream, account *model.Account) error {
	return base.ErrNotImplement
}

var _ base.Driver = (*Onedrive)(nil)