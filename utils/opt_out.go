package utils

import (
	"reflect"
	"strings"
)

var keywords = [...]string{
	"atop",
	"block",
	"dtop",
	"kys",
	"ops",
	"srop",
	"stip",
	"sto",
	"stoo",
	"stoop",
	"store",
	"stp",
	"syop",
	"alisin",
	"annoying",
	"asshole",
	"assholes",
	"bitch",
	"bitches",
	"bloquear",
	"bullshit",
	"bỏ",
	"cancel",
	"cancelar",
	"cunt",
	"cunts",
	"damn",
	"detener",
	"dick",
	"dicks",
	"die",
	"duck",
	"ducking",
	"dừng",
	"eliminar",
	"end",
	"exit",
	"faggot",
	"fuck",
	"fucker",
	"fuckers",
	"fucking",
	"gfy",
	"harass",
	"harassed",
	"harassment",
	"hủy",
	"illegal",
	"itigil",
	"itigillahat",
	"jesus",
	"jfc",
	"joder",
	"kanselahin",
	"kết",
	"lumabas",
	"mag-exit",
	"mag-unsubscribe",
	"mierda",
	"motherfucker",
	"motherfuckers",
	"parar",
	"pendeja",
	"pendejo",
	"phuck",
	"piss",
	"pussies",
	"pussy",
	"puta",
	"puto",
	"quit",
	"remove",
	"retard",
	"retards",
	"salir",
	"shit",
	"shut",
	"sop",
	"spam",
	"spammer",
	"spamming",
	"stop",
	"stopall",
	"suck",
	"terminar",
	"thoát",
	"thúc",
	"unsolicited",
	"unsubscribe",
	"unsubscribed",
	"wakasan",
	"xóa",
	"đăng",
	"停止",
	"全部停止",
	"删除",
	"取消",
	"结束",
	"退出",
	"退订",
	"개",
	"구독취소",
	"꺼져",
	"끝",
	"나가기",
	"망할",
	"모두중지",
	"제거",
	"종료",
	"중지",
	"취소",
}

var phrases = []string{
	"take me off",
	"opted out",
	"lose my number",
	"go away",
	"opt out",
	"do not send",
}

const OptOutMessageBackKey = "opt_out_message_back"
const OptOutDefaultMessageBack = "If this is an emergency, call 911. For more help from CCL contact support@communityconnectlabs.com. Msg freq. varies. Reply STOP to cancel."
const OptOutDisabled = "opt_out_disabled"

// CheckOptOutKeywordPresence is used to check the text contains opt-out words
func CheckOptOutKeywordPresence(text string) bool {
	lowerText := strings.ToLower(text)
	textWords := strings.Split(lowerText, " ")
	checkWords := make([]string, len(textWords))

	for _, word := range textWords {
		newWord := strings.ReplaceAll(word, "?", "")
		newWord = strings.ReplaceAll(newWord, "!", "")
		newWord = strings.ReplaceAll(newWord, ".", "")
		newWord = strings.ReplaceAll(newWord, ",", "")
		checkWords = append(checkWords, newWord)
	}

	return len(intersection(checkWords, keywords)) > 0 || StringSliceContains(phrases, lowerText, false)
}

func intersection(a interface{}, b interface{}) []interface{} {
	set := make([]interface{}, 0)
	av := reflect.ValueOf(a)

	for i := 0; i < av.Len(); i++ {
		el := av.Index(i).Interface()
		if contains(b, el) {
			set = append(set, el)
		}
	}

	return set
}

func contains(a interface{}, e interface{}) bool {
	v := reflect.ValueOf(a)

	for i := 0; i < v.Len(); i++ {
		if v.Index(i).Interface() == e {
			return true
		}
	}
	return false
}

// StringSliceContains determines whether the given slice of strings contains the given string
func StringSliceContains(slice []string, str string, caseSensitive bool) bool {
	for _, s := range slice {
		if (caseSensitive && s == str) || (!caseSensitive && strings.EqualFold(s, str)) {
			return true
		}
	}
	return false
}
