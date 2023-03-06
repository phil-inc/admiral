package gchat

import (
	"net/http"

	"github.com/phil-inc/admiral/pkg/utils"
)

type Builder struct{
	url string
	client *http.Client
	textChannel chan string
	errChannel chan error
}

// New returns a builder for the gchat struct.
func New() *Builder {
	return &Builder {}
}

// Url sets the gchat url.
func (b *Builder) Url(url string) *Builder {
	b.url = url
	return b
}

// TextChannel sets the channel from where
// gchat will take messages.
func (b *Builder) TextChannel(textChannel chan string) *Builder {
	b.textChannel = textChannel
	return b
}

// ErrChannel sets the channel where gchat
// will send its errors.
func (b *Builder) ErrChannel(errChannel chan error) *Builder {
	b.errChannel = errChannel
	return b
}

// Client sets the HTTP client.
func (b *Builder) Client(client *http.Client) *Builder {
	b.client = client
	return b
}

// Build returns a configured gchat struct.
func (b *Builder) Build() *gchat {
	return &gchat {
		url: b.url,
		client: b.client,
		textChannel: b.textChannel,
		errChannel: b.errChannel,
	}
}

type gchat struct {
	url string
	client *http.Client
	textChannel chan string
	errChannel chan error
}

type gchatDTO struct {
	Text string
}

// Stream waits to receive something
// on textChannel, then POSTs it to gchat.
func (g *gchat) Stream() {
	for msg := range g.textChannel {
		dto := msgToDTO(msg)	

		err := utils.Send(dto, "POST", g.url, g.client)
		if err != nil {
			g.errChannel <- err
		}
	}
}

func msgToDTO(text string) *gchatDTO {
	return &gchatDTO{
		Text: text,
	}
}

// Close closes the textChannel. Anything
// already on the stack will get processed.
func (g *gchat) Close() {
	close(g.textChannel)
}

