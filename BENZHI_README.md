# BENZHI_README

## 项目说明
- 项目：benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a
- 项目用途：已完整实现古籍修复方案审议工作台，覆盖损伤建档、完整性校验、专业评估、方案编制、多人同行审议、小样核验、批准封存、状态时间线和 JSON 审计包，并提供原生浏览器工作台、追加事件日志、原子快照、乐观并发与请求幂等。
- Go 工具链：`golang:1.22`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 项目描述
- 项目名称：古籍修复方案审议工作台
- 项目概述：面向古籍保护机构的修复方案审议应用，将单件古籍从损伤建档、专业评估、方案编制、同行审议、小样核验推进到批准封存，并保留完整、可追溯的状态变更与证据记录。
- 核心工作流：修复师创建古籍损伤档案并提交评估，负责人确认保护目标后进入方案编制，修复师提交材料、工序和风险控制方案供同行审议，审议通过后登记小样试验结果，负责人核验全部证据并将方案批准封存。
- 对外接口：由 Go HTTP 服务提供原生 HTML、CSS 和 JavaScript 的浏览器工作台，包含案件列表、状态时间线、损伤评估表、方案编辑器、审议面板、小样核验表和封存摘要；监听地址支持 -addr=127.0.0.1:<port>，默认 127.0.0.1:19081，并在 PORT 为端口号时绑定 127.0.0.1:<PORT>，不得默认绑定 0.0.0.0 或常见低位端口。

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...
cd /app && GOTOOLCHAIN=local go run ./cmd/server -selftest -addr=127.0.0.1:19081
cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a-arm64 linux/arm64
docker run -it benzhi-project-f6c59001-e4b6-493c-ae32-5a2a3be7a53a-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/server -selftest -addr=127.0.0.1:19081`
