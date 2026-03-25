# Cloudflare Worker 部署指南

本指南将指导您如何编写并部署一个 Cloudflare Worker，使其绑定到域名 `api.aiarchlab.xyz`，并将接收到的请求安全地转发到第三方的免费 LLM API（如 SiliconFlow 或 Groq）。

## 前提条件

1. 拥有 Cloudflare 账号
2. 拥有域名 `api.aiarchlab.xyz` 并已在 Cloudflare 上托管
3. 已获取第三方 LLM API 的 API Key（如 SiliconFlow 或 Groq）

## 步骤 1: 创建 Cloudflare Worker

1. 登录 Cloudflare 控制台
2. 点击左侧菜单中的 "Workers 和 Pages"
3. 点击 "创建应用程序"
4. 选择 "Worker"
5. 输入 Worker 名称（例如 "llm-proxy"）
6. 点击 "部署"

## 步骤 2: 编写 Worker 代码

1. 在 Worker 编辑页面，替换默认代码为以下内容：

```javascript
// Cloudflare Worker 代码
addEventListener('fetch', event => {
  event.respondWith(handleRequest(event.request));
});

async function handleRequest(request) {
  // 解析请求 URL
  const url = new URL(request.url);
  
  // 提取路径
  const path = url.pathname;
  
  // 第三方 API 配置
  const API_CONFIG = {
    // SiliconFlow API 配置
    siliconflow: {
      baseUrl: 'https://api.siliconflow.cn/v1/chat/completions',
      apiKey: 'YOUR_SILICONFLOW_API_KEY' // 替换为您的 SiliconFlow API Key
    },
    // Groq API 配置
    groq: {
      baseUrl: 'https://api.groq.com/openai/v1/chat/completions',
      apiKey: 'YOUR_GROQ_API_KEY' // 替换为您的 Groq API Key
    }
  };
  
  // 默认使用 SiliconFlow
  const provider = 'siliconflow';
  const config = API_CONFIG[provider];
  
  // 构建目标 URL
  const targetUrl = config.baseUrl;
  
  // 克隆请求
  const headers = new Headers(request.headers);
  
  // 添加 API Key
  headers.set('Authorization', `Bearer ${config.apiKey}`);
  
  // 移除可能导致问题的头
  headers.delete('Host');
  headers.delete('Connection');
  
  // 构建新请求
  const newRequest = new Request(targetUrl, {
    method: request.method,
    headers: headers,
    body: request.body
  });
  
  // 转发请求
  try {
    const response = await fetch(newRequest);
    
    // 克隆响应
    const responseHeaders = new Headers(response.headers);
    
    // 添加 CORS 头
    responseHeaders.set('Access-Control-Allow-Origin', '*');
    responseHeaders.set('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
    responseHeaders.set('Access-Control-Allow-Headers', 'Content-Type, Authorization');
    
    // 返回响应
    return new Response(response.body, {
      status: response.status,
      statusText: response.statusText,
      headers: responseHeaders
    });
  } catch (error) {
    // 处理错误
    return new Response(JSON.stringify({ error: error.message }), {
      status: 500,
      headers: {
        'Content-Type': 'application/json',
        'Access-Control-Allow-Origin': '*'
      }
    });
  }
}
```

2. 替换代码中的 `YOUR_SILICONFLOW_API_KEY` 和 `YOUR_GROQ_API_KEY` 为您的实际 API Key。

3. 点击 "保存并部署"。

## 步骤 3: 绑定域名

1. 在 Cloudflare 控制台中，点击左侧菜单中的 "Workers 和 Pages"
2. 找到您创建的 Worker，点击它
3. 点击 "触发器"
4. 点击 "添加自定义域"
5. 输入 `api.aiarchlab.xyz`
6. 点击 "添加域"

## 步骤 4: 配置 DNS

1. 在 Cloudflare 控制台中，点击左侧菜单中的 "DNS"
2. 确保 `api.aiarchlab.xyz` 的 DNS 记录已正确配置，指向 Cloudflare Worker

## 步骤 5: 测试

1. 使用 curl 或 Postman 发送测试请求：

```bash
curl -X POST https://api.aiarchlab.xyz/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v3",
    "messages": [
      {
        "role": "user",
        "content": "Hello, world!"
      }
    ]
  }'
```

2. 如果配置正确，您应该会收到来自第三方 LLM API 的响应。

## 安全注意事项

1. 不要在 Worker 代码中硬编码 API Key，建议使用 Cloudflare Worker 的环境变量来存储敏感信息。
2. 考虑添加请求速率限制，防止滥用。
3. 考虑添加身份验证机制，确保只有授权用户才能访问您的 API。

## 故障排查

1. 检查 Cloudflare Worker 的日志，查看是否有错误信息。
2. 确保第三方 API Key 是正确的。
3. 确保域名 `api.aiarchlab.xyz` 已正确绑定到 Worker。
4. 确保 DNS 记录已正确配置。

## 进阶配置：拦截特定邮件前缀并触发 Webhook

### 步骤 1: 配置 Cloudflare Email Routing

1. 登录 Cloudflare 控制台
2. 点击左侧菜单中的 "电子邮件"
3. 点击 "Email Routing"
4. 确保已启用 Email Routing 功能
5. 点击 "创建路由规则"

### 步骤 2: 创建邮件路由规则

1. 在 "收件人" 字段中输入 `dev_*@aiarchlab.xyz`（或您想要拦截的邮件前缀模式）
2. 在 "操作" 中选择 "发送到 Worker"
3. 选择您之前创建的 Worker
4. 点击 "保存"

### 步骤 3: 修改 Worker 代码以处理邮件

```javascript
// Cloudflare Worker 代码（支持邮件处理）
addEventListener('fetch', event => {
  event.respondWith(handleRequest(event.request));
});

addEventListener('email', event => {
  event.respondWith(handleEmail(event.email));
});

async function handleRequest(request) {
  // 之前的代码...
}

async function handleEmail(email) {
  // 解析邮件内容
  const { from, to, subject, text, html } = email;
  
  // 提取验证码
  const code = extractVerificationCode(text);
  
  // 触发 Webhook
  if (code) {
    await fetch('https://your-webhook-url.com', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        email: to,
        code: code,
        from: from,
        subject: subject
      })
    });
  }
  
  // 返回空响应
  return new Response();
}

function extractVerificationCode(text) {
  // 匹配6位数字
  const match = text.match(/\b\d{6}\b/);
  return match ? match[0] : null;
}
```

### 步骤 4: 测试邮件拦截

1. 发送一封邮件到 `dev_test@aiarchlab.xyz`
2. 查看 Cloudflare Worker 的日志，确认邮件被正确处理
3. 检查 Webhook 是否收到了验证码信息

### 注意事项

1. 确保您的 Worker 有足够的执行时间来处理邮件
2. 考虑添加错误处理和重试机制
3. 确保 Webhook 端点是安全的，避免被滥用