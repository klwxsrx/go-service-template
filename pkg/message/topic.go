package message

import (
	"fmt"
	"strings"

	pkgstrings "github.com/klwxsrx/go-service-template/pkg/strings"
)

type (
	Topic              string
	TopicBuilderOption func(*topicBuilder)

	topicBuilder struct {
		baseName    string
		domain      string
		aggregate   string
		messageType string
		customTags  []string
	}
)

func (b *topicBuilder) Build() Topic {
	const separator = '.'

	sb := strings.Builder{}
	sb.WriteString(b.baseName)

	addTagIfNotEmpty := func(tag string) {
		if tag != "" {
			sb.WriteRune(separator)
			sb.WriteString(tag)
		}
	}

	addTagIfNotEmpty(b.domain)
	addTagIfNotEmpty(b.aggregate)
	addTagIfNotEmpty(b.messageType)
	for _, tag := range b.customTags {
		addTagIfNotEmpty(tag)
	}

	return Topic(sb.String())
}

func WithTopicDomainName(name string) TopicBuilderOption {
	name = pkgstrings.ToKebabCase(name)
	return func(builder *topicBuilder) {
		builder.domain = fmt.Sprintf("%s-domain", name)
	}
}

func WithTopicAggregateName(name string) TopicBuilderOption {
	name = pkgstrings.ToKebabCase(name)
	return func(builder *topicBuilder) {
		builder.aggregate = fmt.Sprintf("%s-aggregate", name)
	}
}

func WithTopicMessageType(msgType string) TopicBuilderOption {
	msgType = pkgstrings.ToKebabCase(msgType)
	return func(builder *topicBuilder) {
		builder.messageType = fmt.Sprintf("%s-type", msgType)
	}
}

func WithTopicCustomTags(tags ...string) TopicBuilderOption {
	for i := range tags {
		tags[i] = pkgstrings.ToKebabCase(tags[i])
	}

	return func(builder *topicBuilder) {
		builder.customTags = append(builder.customTags, tags...)
	}
}

func NewTopic(baseName string, opts ...TopicBuilderOption) Topic {
	builder := topicBuilder{baseName: pkgstrings.ToKebabCase(baseName)}
	for _, opt := range opts {
		opt(&builder)
	}

	return builder.Build()
}

func NewRawTopic(topic string) Topic {
	return Topic(topic)
}
