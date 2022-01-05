package rapidpro

import "testing"

func Test_checkOptOutKeywordPresence(t *testing.T) {
	type args struct {
		text string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "Friend", args: args{text: "Friend"}, want: false},
		{name: "Stop All", args: args{text: "All"}, want: false},
		{name: "Stop", args: args{text: "Stop"}, want: true},
		{name: "Mother", args: args{text: "Mother"}, want: false},
		{name: "Legal", args: args{text: "legal"}, want: false},
		{name: "Exit", args: args{text: "exit"}, want: true},
		{name: "Fuck you", args: args{text: "Fuck you"}, want: true},
		{name: "Suck", args: args{text: "It is a suck"}, want: true},
		{name: "What a fuck?", args: args{text: "What a fuck?"}, want: true},
		{name: "What a fuck.", args: args{text: "What a fuck."}, want: true},
		{name: "Hey, please stop! I do not want it anymore", args: args{
			text: "Hey, please stop! I do not want it anymore",
		}, want: true},
		{name: "全部停止", args: args{text: "全部停止"}, want: true},
		{name: "구독취소", args: args{text: "구독취소"}, want: true},
		{name: "취소", args: args{text: "취소"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkOptOutKeywordPresence(tt.args.text); got != tt.want {
				t.Errorf("checkOptOutKeywordPresence() = %v, want %v", got, tt.want)
			}
		})
	}
}
