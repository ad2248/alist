package model

import (
	"github.com/Xhofe/alist/conf"
	"github.com/robfig/cron/v3"
	"time"
)

type Account struct {
	ID             uint   `json:"id" gorm:"primaryKey"`                  // 唯一ID
	Name           string `json:"name" gorm:"unique" binding:"required"` // 唯一名称
	Index          int    `json:"index"`                                 // 序号 用于排序
	Type           string `json:"type"`                                  // 类型，即driver
	Username       string `json:"username"`
	Password       string `json:"password"`
	RefreshToken   string `json:"refresh_token"`
	AccessToken    string `json:"access_token"`
	RootFolder     string `json:"root_folder"`
	Status         string `json:"status"` // 状态
	CronId         int
	DriveId        string
	Limit          int        `json:"limit"`
	OrderBy        string     `json:"order_by"`
	OrderDirection string     `json:"order_direction"`
	UpdatedAt      *time.Time `json:"updated_at"`
	Search         bool       `json:"search"`
	ClientId       string     `json:"client_id"`
	ClientSecret   string     `json:"client_secret"`
	Zone           string     `json:"zone"`
	RedirectUri    string     `json:"redirect_uri"`
	SiteUrl        string     `json:"site_url"`
	SiteId         string     `json:"site_id"`
	InternalType   string     `json:"internal_type"`
	WebdavProxy    bool       `json:"webdav_proxy"`
	Proxy          bool       `json:"proxy"`       // 是否中转
	//AllowProxy     bool       `json:"allow_proxy"` // 是否允许中转下载
	ProxyUrl       string     `json:"proxy_url"`   // 用于中转下载服务的URL
}

var accountsMap = map[string]Account{}

// SaveAccount save account to database
func SaveAccount(account *Account) error {
	if err := conf.DB.Save(account).Error; err != nil {
		return err
	}
	RegisterAccount(*account)
	return nil
}

func CreateAccount(account *Account) error {
	if err := conf.DB.Create(account).Error; err != nil {
		return err
	}
	RegisterAccount(*account)
	return nil
}

func DeleteAccount(id uint) error {
	var account Account
	account.ID = id
	if err := conf.DB.First(&account).Error; err != nil {
		return err
	}
	name := account.Name
	conf.Cron.Remove(cron.EntryID(account.CronId))
	if err := conf.DB.Delete(&account).Error; err != nil {
		return err
	}
	delete(accountsMap, name)
	return nil
}

func DeleteAccountFromMap(name string) {
	delete(accountsMap, name)
}

func AccountsCount() int {
	return len(accountsMap)
}

func RegisterAccount(account Account) {
	accountsMap[account.Name] = account
}

func GetAccount(name string) (Account, bool) {
	if len(accountsMap) == 1 {
		for _, v := range accountsMap {
			return v, true
		}
	}
	account, ok := accountsMap[name]
	return account, ok
}

func GetAccountById(id uint) (*Account, error) {
	var account Account
	account.ID = id
	if err := conf.DB.First(&account).Error; err != nil {
		return nil, err
	}
	return &account, nil
}

func GetAccountFiles() ([]File, error) {
	files := make([]File, 0)
	var accounts []Account
	if err := conf.DB.Order("`index`").Find(&accounts).Error; err != nil {
		return nil, err
	}
	for _, v := range accounts {
		files = append(files, File{
			Name:      v.Name,
			Size:      0,
			Type:      conf.FOLDER,
			UpdatedAt: v.UpdatedAt,
		})
	}
	return files, nil
}

func GetAccounts() ([]Account, error) {
	var accounts []Account
	if err := conf.DB.Order("`index`").Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}
