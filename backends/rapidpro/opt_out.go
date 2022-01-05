package rapidpro

import (
	"strings"
	"reflect"
)

var keywords = [...]string{
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
	"harass",
	"harassed",
	"hủy",
	"illegal",
	"itigil",
	"itigillahat",
	"jesus",
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

func checkOptOutKeywordPresence(text string) bool {
	textWords := strings.Split(strings.ToLower(text), " ")
	checkWords := make([]string, len(textWords))

	for _, word := range textWords {
		newWord := strings.ReplaceAll(word, "?", "")
		newWord = strings.ReplaceAll(newWord, "!", "")
		newWord = strings.ReplaceAll(newWord, ".", "")
		newWord = strings.ReplaceAll(newWord, ",", "")
		checkWords = append(checkWords, newWord)
	}

	return len(intersection(checkWords, keywords)) > 0
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
