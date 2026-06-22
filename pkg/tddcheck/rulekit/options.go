package rulekit

type Option func(*RuleOptions)

type RuleOptions struct {
	Root   string
	Config Config
}

func NewRuleOptions(root string, options ...Option) RuleOptions {
	values := RuleOptions{Root: root}
	for _, option := range options {
		option(&values)
	}
	return values
}

func WithConfig(config Config) Option {
	return func(options *RuleOptions) {
		options.Config = config
	}
}
