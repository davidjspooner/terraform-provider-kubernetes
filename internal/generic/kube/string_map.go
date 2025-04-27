package kube

import (
	"encoding/base64"
	"fmt"
	"strings"
)

type StringMap struct {
	textData          map[string]string
	base64EncodedData map[string]string
}

func (sm *StringMap) Clear() {
	sm.textData = nil
	sm.base64EncodedData = nil
}

func (sm *StringMap) validateUniqueKey(key string) error {
	if key == "" {
		return fmt.Errorf("key is empty")
	}
	for _, badChar := range "/\\?%*:|\"<>\n\r\t\b\f" {
		if strings.ContainsRune(key, badChar) {
			return fmt.Errorf("key %q contains %q", key, badChar)
		}
	}
	lowerKey := strings.ToLower(key)
	for exisingKey := range sm.textData {
		if lowerKey == strings.ToLower(exisingKey) {
			return fmt.Errorf("key %q already exists", key)
		}
	}
	for exisingKey := range sm.base64EncodedData {
		if lowerKey == strings.ToLower(exisingKey) {
			return fmt.Errorf("key %q already exists", key)
		}
	}
	return nil
}

func (sm *StringMap) AddText(key, value string) error {
	if err := sm.validateUniqueKey(key); err != nil {
		return err
	}
	if sm.textData == nil {
		sm.textData = make(map[string]string)
	}
	sm.textData[key] = value
	return nil
}
func (sm *StringMap) AddBase64(key, value string) error {
	if err := sm.validateUniqueKey(key); err != nil {
		return err
	}
	if sm.base64EncodedData == nil {
		sm.base64EncodedData = make(map[string]string)
	}
	_, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return fmt.Errorf("value for key %q is not base64: %w", key, err)
	}
	sm.base64EncodedData[key] = value
	return nil
}
func (sm *StringMap) EncodeAsBase64AndAdd(key string, value []byte) error {
	if err := sm.validateUniqueKey(key); err != nil {
		return err
	}
	if sm.base64EncodedData == nil {
		sm.base64EncodedData = make(map[string]string)
	}
	base64Value := base64.StdEncoding.EncodeToString(value)
	sm.base64EncodedData[key] = base64Value
	return nil
}
func (sm *StringMap) GetUnstructuredText() map[string]interface{} {
	if sm.textData == nil {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range sm.textData {
		result[k] = v
	}
	return result
}
func (sm *StringMap) GetUnstructuredBase64() map[string]interface{} {
	if sm.base64EncodedData == nil {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range sm.base64EncodedData {
		result[k] = v
	}
	return result
}
func (sm *StringMap) ForEachTextContent(f func(key string, value string) error) error {
	if sm.textData == nil {
		return nil
	}
	for k, v := range sm.textData {
		if err := f(k, v); err != nil {
			return err
		}
	}
	return nil
}

func (sm *StringMap) ForEachBase64Content(f func(key string, value string) error) error {
	if sm.base64EncodedData == nil {
		return nil
	}
	for k, v := range sm.base64EncodedData {
		if err := f(k, v); err != nil {
			return err
		}
	}
	return nil
}
