package message

import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
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
	name = strcase.ToKebab(name)
	return func(builder *topicBuilder) {
		builder.domain = fmt.Sprintf("%s-domain", name)
	}
}

func WithTopicAggregateName(name string) TopicBuilderOption {
	name = strcase.ToKebab(name)
	return func(builder *topicBuilder) {
		builder.domain = fmt.Sprintf("%s-aggregate", name)
	}
}

func WithTopicMessageType(msgType string) TopicBuilderOption {
	msgType = strcase.ToKebab(msgType)
	return func(builder *topicBuilder) {
		builder.domain = fmt.Sprintf("%s-type", msgType)
	}
}

func WithTopicCustomTags(tags ...string) TopicBuilderOption {
	for i := 0; i < len(tags); i++ {
		tags[i] = strcase.ToKebab(tags[i])
	}

	return func(builder *topicBuilder) {
		builder.customTags = append(builder.customTags, tags...)
	}
}

func NewTopic(baseName string, opts ...TopicBuilderOption) Topic {
	builder := topicBuilder{baseName: strcase.ToKebab(baseName)}
	for _, opt := range opts {
		opt(&builder)
	}

	return builder.Build()
}

func NewRawTopic(topic string) Topic {
	return Topic(topic)
}
