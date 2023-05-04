package mgage

import "github.com/nyaruka/courier"

type EmptyMGAChannel struct{}

func (e EmptyMGAChannel) ID() courier.ChannelID                  { return courier.NilChannelID }
func (e EmptyMGAChannel) UUID() courier.ChannelUUID              { return courier.NilChannelUUID }
func (e EmptyMGAChannel) Name() string                           { return "" }
func (e EmptyMGAChannel) ChannelType() courier.ChannelType       { return "MGA" }
func (e EmptyMGAChannel) Schemes() []string                      { return []string{"tel"} }
func (e EmptyMGAChannel) Country() string                        { return "" }
func (e EmptyMGAChannel) Address() string                        { return "" }
func (e EmptyMGAChannel) ChannelAddress() courier.ChannelAddress { return "" }
func (e EmptyMGAChannel) Roles() []courier.ChannelRole           { return nil }

func (e EmptyMGAChannel) IsScheme(_ string) bool                                           { return false }
func (e EmptyMGAChannel) CallbackDomain(_ string) string                                   { return "" }
func (e EmptyMGAChannel) ConfigForKey(_ string, _ interface{}) interface{}                 { return nil }
func (e EmptyMGAChannel) StringConfigForKey(key string, defaultValue string) string        { return "" }
func (e EmptyMGAChannel) BoolConfigForKey(key string, defaultValue bool) bool              { return false }
func (e EmptyMGAChannel) IntConfigForKey(key string, defaultValue int) int                 { return 0 }
func (e EmptyMGAChannel) OrgConfigForKey(key string, defaultValue interface{}) interface{} { return nil }
