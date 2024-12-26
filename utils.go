package courier

import (
	"github.com/nyaruka/courier/utils"
)

// GetOptOutMessage used to get the translated opt-out message
func GetOptOutMessage(channel Channel, contact Contact) interface{} {
	language, _ := contact.Language().Value()
	defaultOrgOptOutMessege := channel.OrgConfigForKey(utils.OptOutMessageBackKey, utils.OptOutDefaultMessageBack)
	orgOptOutMessageI18n := channel.OrgConfigForKey(utils.OptOutMessageBackI18n, nil)
	if orgOptOutMessageI18n != nil && language != nil {
		if orgOptOutMessageI18nDict, ok := orgOptOutMessageI18n.(map[string]interface{}); ok {
			if translatedMsg, ok := orgOptOutMessageI18nDict[language.(string)]; ok {
				return translatedMsg
			}
		}
	}
	return defaultOrgOptOutMessege
}
