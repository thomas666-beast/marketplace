package i18n

import (
    "encoding/json"
    "fmt"

    "github.com/nicksnyder/go-i18n/v2/i18n"
    "golang.org/x/text/language"
)

type Bundle struct {
    *i18n.Bundle
}

func NewBundle() (*Bundle, error) {
    b := i18n.NewBundle(language.English)
    b.RegisterUnmarshalFunc("json", json.Unmarshal)

    entries, err := localesFS.ReadDir("locales")
    if err != nil {
        return nil, fmt.Errorf("read embedded locales: %w", err)
    }

    loaded := 0
    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }
        path := "locales/" + entry.Name()
        data, err := localesFS.ReadFile(path)
        if err != nil {
            return nil, fmt.Errorf("read %s: %w", path, err)
        }
        if _, err := b.ParseMessageFileBytes(data, entry.Name()); err != nil {
            return nil, fmt.Errorf("parse %s: %w", path, err)
        }
        loaded++
    }

    if loaded == 0 {
        return nil, fmt.Errorf("no locale files embedded")
    }

    return &Bundle{Bundle: b}, nil
}

func (b *Bundle) Translate(locale, messageID string) string {
    localizer := i18n.NewLocalizer(b.Bundle, locale, "en")
    msg, err := localizer.Localize(&i18n.LocalizeConfig{
        MessageID: messageID,
    })
    if err != nil {
        return messageID
    }
    return msg
}
