# OpenAI Integration Setup Guide

This guide explains how to configure and set up OpenAI for real AI chat functionality in the LoveGuru application.

## Overview

The dummy AI responses have been replaced with real OpenAI integration. The system now:

- ✅ Generates real AI responses using OpenAI GPT models
- ✅ Supports both synchronous chat and streaming responses
- ✅ Stores AI interactions in the database for analytics
- ✅ Provides proper error handling and fallbacks
- ✅ Configurable model, token limits, and API settings
- ✅ Professional love advice counselor persona

## Prerequisites

1. **OpenAI Account**: Create an account at [OpenAI Platform](https://platform.openai.com)
2. **API Key**: Generate an API key from the OpenAI dashboard
3. **Billing**: Ensure you have billing set up for API usage

## Configuration

### 1. Environment Variables

Add the following environment variables to your `.env` file:

```bash
# OpenAI Configuration
OPENAI_API_KEY=sk-your-openai-api-key-here
OPENAI_BASE_URL=https://api.openai.com
OPENAI_MODEL=gpt-3.5-turbo
OPENAI_MAX_TOKENS=500
```

### 2. Config File

Alternatively, add to your `config.yaml` or `config.yml`:

```yaml
openai:
  api_key: "sk-your-openai-api-key-here"
  base_url: "https://api.openai.com"
  model: "gpt-3.5-turbo"
  max_tokens: 500
```

### 3. Application Configuration

The system automatically loads OpenAI configuration from environment variables or config files. The configuration includes:

- `api_key`: Your OpenAI API key
- `base_url`: OpenAI API base URL (default: https://api.openai.com)
- `model`: GPT model to use (default: gpt-3.5-turbo)
- `max_tokens`: Maximum tokens for response (default: 500)

## Testing the Integration

### 1. Validate Configuration

Run the server and check for this warning message:
```
Warning: OpenAI API key not configured. AI chat functionality will not work.
```

If you see this, ensure your OpenAI API key is properly configured.

### 2. Test Chat API

Make a gRPC call to test the chat functionality:

```bash
grpcurl -d '{"message": "I need advice about my relationship", "context": "romantic_advice"}' \
  -H 'Authorization: Bearer your-jwt-token' \
  localhost:50051 loveguru.ai.AIService/Chat
```

**Expected Response:**
```json
{
  "response": "I'd be happy to help with your relationship questions. As a professional love advisor, I can provide guidance on communication, trust, intimacy, and navigating challenges in your relationship. What specific aspect would you like to discuss?"
}
```

### 3. Test Chat Stream

Test the streaming chat functionality:

```bash
grpcurl -d '{"message": "How do I know if my partner loves me?", "context": "relationship_advice"}' \
  -H 'Authorization: Bearer your-jwt-token' \
  localhost:50051 loveguru.ai.AIService/ChatStream
```

## Integration Details

### What's Changed

1. **Removed Dummy Implementation**:
   - ✅ Removed `callAI()` dummy method
   - ✅ Fixed `ChatStream()` to use real OpenAI API instead of echo
   - ✅ Removed hardcoded dummy responses

2. **Added Real OpenAI Integration**:
   - ✅ Real API calls to OpenAI GPT models
   - ✅ Configurable model selection (gpt-3.5-turbo, gpt-4, etc.)
   - ✅ Proper token management and rate limiting
   - ✅ Professional love counselor persona
   - ✅ Context-aware responses with conversation history

3. **Enhanced Configuration**:
   - ✅ Integrated with application config system
   - ✅ Environment variable support
   - ✅ Configurable model parameters
   - ✅ Startup validation and warnings

### Service Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   gRPC Client   │    │   AI Service    │    │ OpenAI Client   │
│                 │    │                 │    │                 │
│      Chat       │───▶│      Chat       │───▶│   Chat API      │
│                 │    │                 │    │                 │
│   ChatStream    │───▶│   ChatStream    │───▶│  Stream API     │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │                        │
                              ▼                        ▼
                       ┌─────────────────┐    ┌─────────────────┐
                       │   Database      │    │  OpenAI Cloud   │
                       │                 │    │                 │
                       │ AI Interactions │    │   GPT Models    │
                       │   User Data     │    │  Token Limits   │
                       └─────────────────┘    └─────────────────┘
```

## API Usage

### Chat Endpoint

```protobuf
message ChatRequest {
  string message = 1;
  string context = 2; // relationship type, previous session id
  string session_id = 3; // optional session ID for chat context
}

message ChatResponse {
  string response = 1;
}
```

### ChatStream Endpoint

```protobuf
message ChatMessage {
  string message = 1;
  string context = 2;
}

// Bi-directional streaming for real-time chat
rpc ChatStream (stream ChatMessage) returns (stream ChatMessage);
```

## Frontend Integration

### JavaScript Example

```javascript
import { grpc } from '@improbable-eng/grpc-web';

// Create gRPC client
const aiClient = grpc.client(loveguru.ai.AIService.Chat, {
  host: 'http://localhost:50051'
});

// Send chat request
const chatRequest = {
  message: "I need relationship advice",
  context: "romantic_advice",
  session_id: "session-123"
};

aiClient.send(chatRequest);
aiClient.finishSend();

// Handle response
aiClient.onMessage((response) => {
  console.log('AI Response:', response.response);
});
```

### WebSocket Alternative

For real-time streaming, you can also implement WebSocket connections:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws/ai-chat');

ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'chat',
    message: 'How do I fix my relationship?',
    context: 'couples_counseling'
  }));
};

ws.onmessage = (event) => {
  const response = JSON.parse(event.data);
  console.log('Streaming AI Response:', response.response);
};
```

## Error Handling

### Common Issues

1. **"OpenAI API key not configured"**
   - Check environment variables are set correctly
   - Verify config.yaml file syntax
   - Ensure API key is valid and has billing setup

2. **API Rate Limits**
   - OpenAI has rate limits per minute
   - Implement exponential backoff for retries
   - Consider upgrading your OpenAI plan for higher limits

3. **Token Limits**
   - Responses truncated if exceeding max_tokens
   - Context conversations use more tokens
   - Consider implementing conversation summaries

### Fallback Strategy

```go
// In production, implement fallback logic
if openaiResp, err := s.openai.Chat(ctx, prompt, context); err != nil {
    // Log error but provide helpful fallback
    return "I apologize, but I'm having trouble processing your request right now. Please try again in a moment, or feel free to rephrase your question.", nil
}
```

## Monitoring and Analytics

### Database Tracking

All AI interactions are automatically stored in the database:

```sql
SELECT * FROM ai_interactions 
WHERE user_id = 'user-uuid' 
ORDER BY created_at DESC 
LIMIT 10;
```

### Key Metrics to Monitor

1. **Response Times**: Track API latency
2. **Error Rates**: Monitor failed API calls
3. **Token Usage**: Track costs and usage patterns
4. **User Engagement**: Conversation lengths and topics

### Health Checks

```bash
# Test API connectivity
curl -X POST https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"

# Check application health
curl http://localhost:8080/health
```

## Security Considerations

1. **API Key Security**: Never expose API keys in frontend code
2. **Rate Limiting**: Implement client-side rate limiting
3. **Input Sanitization**: Validate user inputs before API calls
4. **Response Filtering**: Consider content filtering for inappropriate requests
5. **Cost Management**: Implement usage limits and alerts

## Production Deployment

### Environment Setup

```bash
# Production environment variables
OPENAI_API_KEY=sk-prod-your-production-api-key
OPENAI_MODEL=gpt-4  # Upgrade to GPT-4 for better responses
OPENAI_MAX_TOKENS=1000  # Allow longer responses
```

### Scaling Considerations

1. **Caching**: Cache frequent responses to reduce API costs
2. **Rate Limiting**: Implement per-user rate limits
3. **Load Balancing**: Distribute requests across multiple API keys if needed
4. **Monitoring**: Set up alerts for API failures and costs

### Performance Optimization

1. **Connection Pooling**: Reuse HTTP connections
2. **Async Processing**: Process requests asynchronously where possible
3. **Token Optimization**: Use efficient prompting to reduce token usage
4. **Response Streaming**: Use streaming for better user experience

## Troubleshooting

### Common Error Messages

1. **"Unauthorized"**: Invalid API key or missing billing
2. **"Rate limit exceeded"**: Too many requests per minute
3. **"Context length exceeded"**: Conversation too long for model
4. **"Model not found"**: Invalid model name specified

### Debug Commands

```bash
# Test configuration loading
OPENAI_API_KEY=test-key go run cmd/server/main.go

# Monitor API calls
tail -f logs/application.log | grep -i openai

# Test with invalid credentials to verify error handling
OPENAI_API_KEY="" go run cmd/server/main.go
```

## Model Recommendations

### For LoveGuru Use Case

- **gpt-3.5-turbo**: Good balance of cost and quality for most use cases
- **gpt-4**: Higher quality responses but more expensive
- **gpt-4-turbo**: Latest model with improved performance

### Configuration Examples

```yaml
# Cost-effective configuration
openai:
  model: "gpt-3.5-turbo"
  max_tokens: 300

# High-quality configuration  
openai:
  model: "gpt-4"
  max_tokens: 800
```

## Next Steps

1. ✅ **Complete**: Replace dummy AI with real OpenAI integration
2. 🔄 **Optional**: Implement conversation memory and summaries
3. 🔄 **Optional**: Add multiple AI model support (Claude, Gemini)
4. 🔄 **Optional**: Implement conversation analytics dashboard
5. 🔄 **Optional**: Add AI response moderation and filtering

---

**Integration Status**: ✅ Complete

The dummy AI responses have been successfully replaced with real OpenAI integration. The system now provides professional love advice using GPT models and is ready for production use.