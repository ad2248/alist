package alidrive

import (
	"errors"
	"fmt"
	"github.com/Xhofe/alist/conf"
	"github.com/Xhofe/alist/drivers/base"
	"github.com/Xhofe/alist/model"
	"github.com/Xhofe/alist/utils"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"path/filepath"
	"strings"
	"time"
)

var aliClient = resty.New()

type AliRespError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type AliFiles struct {
	Items      []AliFile `json:"items"`
	NextMarker string    `json:"next_marker"`
}

type AliFile struct {
	DriveId       string     `json:"drive_id"`
	CreatedAt     *time.Time `json:"created_at"`
	FileExtension string     `json:"file_extension"`
	FileId        string     `json:"file_id"`
	Type          string     `json:"type"`
	Name          string     `json:"name"`
	Category      string     `json:"category"`
	ParentFileId  string     `json:"parent_file_id"`
	UpdatedAt     *time.Time `json:"updated_at"`
	Size          int64      `json:"size"`
	Thumbnail     string     `json:"thumbnail"`
	Url           string     `json:"url"`
}

func (driver AliDrive) FormatFile(file *AliFile) *model.File {
	f := &model.File{
		Id:        file.FileId,
		Name:      file.Name,
		Size:      file.Size,
		UpdatedAt: file.UpdatedAt,
		Thumbnail: file.Thumbnail,
		Driver:    driver.Config().Name,
		Url:       file.Url,
	}
	if file.Type == "folder" {
		f.Type = conf.FOLDER
	} else {
		f.Type = utils.GetFileType(file.FileExtension)
	}
	if file.Category == "video" {
		f.Type = conf.VIDEO
	}
	if file.Category == "image" {
		f.Type = conf.IMAGE
	}
	return f
}

func (driver AliDrive) GetFiles(fileId string, account *model.Account) ([]AliFile, error) {
	marker := "first"
	res := make([]AliFile, 0)
	for marker != "" {
		if marker == "first" {
			marker = ""
		}
		var resp AliFiles
		var e AliRespError
		_, err := aliClient.R().
			SetResult(&resp).
			SetError(&e).
			SetHeader("authorization", "Bearer\t"+account.AccessToken).
			SetBody(base.Json{
				"drive_id":                account.DriveId,
				"fields":                  "*",
				"image_thumbnail_process": "image/resize,w_400/format,jpeg",
				"image_url_process":       "image/resize,w_1920/format,jpeg",
				"limit":                   account.Limit,
				"marker":                  marker,
				"order_by":                account.OrderBy,
				"order_direction":         account.OrderDirection,
				"parent_file_id":          fileId,
				"video_thumbnail_process": "video/snapshot,t_0,f_jpg,ar_auto,w_300",
				"url_expire_sec":          14400,
			}).Post("https://api.aliyundrive.com/v2/file/list")
		if err != nil {
			return nil, err
		}
		if e.Code != "" {
			if e.Code == "AccessTokenInvalid" {
				err = driver.RefreshToken(account)
				if err != nil {
					return nil, err
				} else {
					_ = model.SaveAccount(account)
					return driver.GetFiles(fileId, account)
				}
			}
			return nil, fmt.Errorf("%s", e.Message)
		}
		marker = resp.NextMarker
		res = append(res, resp.Items...)
	}
	return res, nil
}

func (driver AliDrive) GetFile(path string, account *model.Account) (*AliFile, error) {
	dir, name := filepath.Split(path)
	dir = utils.ParsePath(dir)
	_, err := driver.Files(dir, account)
	if err != nil {
		return nil, err
	}
	parentFiles_, _ := base.GetCache(dir, account)
	parentFiles, _ := parentFiles_.([]AliFile)
	for _, file := range parentFiles {
		if file.Name == name {
			if file.Type == "file" {
				return &file, err
			} else {
				return nil, fmt.Errorf("not file")
			}
		}
	}
	return nil, base.ErrPathNotFound
}

func (driver AliDrive) RefreshToken(account *model.Account) error {
	url := "https://auth.aliyundrive.com/v2/account/token"
	var resp base.TokenResp
	var e AliRespError
	_, err := aliClient.R().
		//ForceContentType("application/json").
		SetBody(base.Json{"refresh_token": account.RefreshToken, "grant_type": "refresh_token"}).
		SetResult(&resp).
		SetError(&e).
		Post(url)
	if err != nil {
		account.Status = err.Error()
		return err
	}
	log.Debugf("%+v,%+v", resp, e)
	if e.Code != "" {
		account.Status = e.Message
		return fmt.Errorf("failed to refresh token: %s", e.Message)
	} else {
		account.Status = "work"
	}
	account.RefreshToken, account.AccessToken = resp.RefreshToken, resp.AccessToken
	return nil
}

func (driver AliDrive) Rename(fileId, name string, account *model.Account) error {
	var resp base.Json
	var e AliRespError
	_, err := aliClient.R().SetResult(&resp).SetError(&e).
		SetHeader("authorization", "Bearer\t"+account.AccessToken).
		SetBody(base.Json{
			"check_name_mode": "refuse",
			"drive_id":        account.DriveId,
			"file_id":         fileId,
			"name":            name,
		}).Post("https://api.aliyundrive.com/v3/file/update")
	if err != nil {
		return err
	}
	if e.Code != "" {
		if e.Code == "AccessTokenInvalid" {
			err = driver.RefreshToken(account)
			if err != nil {
				return err
			} else {
				_ = model.SaveAccount(account)
				return driver.Rename(fileId, name, account)
			}
		}
		return fmt.Errorf("%s", e.Message)
	}
	if resp["name"] == name {
		return nil
	}
	return fmt.Errorf("%+v", resp)
}

func (driver AliDrive) Batch(srcId,dstId string, account *model.Account) error {
	var e AliRespError
	res, err := aliClient.R().SetError(&e).
		SetHeader("authorization", "Bearer\t"+account.AccessToken).
		SetBody(base.Json{
			"requests": []base.Json{
				{
					"headers": base.Json{
						"Content-Type": "application/json",
					},
					"method":"POST",
					"id":srcId,
					"body":base.Json{
						"drive_id": account.DriveId,
						"file_id":srcId,
						"to_drive_id":account.DriveId,
						"to_parent_file_id":dstId,
					},
				},
			},
			"resource": "file",
		}).Post("https://api.aliyundrive.com/v3/batch")
	if err != nil {
		return err
	}
	if e.Code != "" {
		if e.Code == "AccessTokenInvalid" {
			err = driver.RefreshToken(account)
			if err != nil {
				return err
			} else {
				_ = model.SaveAccount(account)
				return driver.Batch(srcId, dstId, account)
			}
		}
		return fmt.Errorf("%s", e.Message)
	}
	if strings.Contains(res.String(), `"status":200`) {
		return nil
	}
	return errors.New(res.String())
}

func init() {
	base.RegisterDriver(&AliDrive{})
	aliClient.
		SetRetryCount(3).
		SetHeader("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36").
		SetHeader("content-type", "application/json").
		SetHeader("origin", "https://www.aliyundrive.com")
}
