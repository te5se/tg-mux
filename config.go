package tgmux

type Localization struct {
	OnError            string
	CommandNotFound    string
	UseStartToRegister string
}

var defaultLocalization = Localization{
	OnError:            "Error occured on the server. Support is notified",
	CommandNotFound:    "Command isn't supported",
	UseStartToRegister: "To use the bot, press /start",
}
