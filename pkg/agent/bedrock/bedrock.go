package bedrock

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
)

type Builder struct {
	region string
	agentID string
	agentAlias string
}

func New() *Builder {
	return &Builder{}
}

func (b *Builder) Region(region string) *Builder {
	b.region = region
	return b
}

func (b *Builder) AgentID(agentID string) *Builder {
	b.agentID = agentID
	return b
}

func (b *Builder) AgentAlias(agentAlias string) *Builder {
	b.agentAlias = agentAlias
	return b
}

func (b *Builder) Build() (*bedrock, error) {
	// Load AWS config -- you should set this up outside of Admiral
	// https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(b.region),
	)
	if err != nil {
		return nil, err
	}

	// Bedrock Agent Runtime client
	client := bedrockagentruntime.NewFromConfig(cfg)

	return &bedrock{
		region: b.region,
		agentID: b.agentID,
		agentAlias: b.agentAlias,
		client: client,
	}, nil
}

type bedrock struct {
	region string
	agentID string
	agentAlias string
	client *bedrockagentruntime.Client
}

func (b *bedrock) Process(input string) (string, error) {
	ctx := context.TODO()

	// Call the Bedrock Agent runtime API
	output, err := b.client.InvokeAgent(ctx, &bedrockagentruntime.InvokeAgentInput{
		AgentId: &b.agentID,
		AgentAliasId: &b.agentAlias,
		InputText: &input,
	})
	if err != nil {
		return "", err
	}

	// Process the response stream
	stream := output.GetStream()
	defer stream.Close()

	var responseBuilder strings.Builder

	for event := range stream.Events() {
		if chunk, ok := event.(*types.ResponseStreamMemberChunk); ok {
			responseBuilder.Write(chunk.Value.Bytes)
		}
	}
	return "", nil
}

func (b *bedrock) Close() error {
	return nil
}
