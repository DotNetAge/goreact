# 重构计划

GoReact的定位是提供基于ReAct理论的智能体编排的框架。

特点：

- 提供ReAct的基本模块以及构建智能体应用的常规模块
    - thinker - 思考相关的功能
    - actor - 执行相关的功能
    - observer - 观察结果相关的功能
    - terminator - 结果整理与循环控制
    - tools - 工具定义与管理
    - agent - 智能体定义与管理
    - metrics - 可观察性接口及基本实现
    - prompt - 提示词工程
    - skill - 技能定义与管理
    - model - 模型定义管理
    - steps - 所有用于组合管线的步骤定义
    - engine - 程序入口与管线组装
    - core - 所有共享对象
    
- 开发人员可以直接引用功能包内类进行二次开发
- 开发人员可以用更简单的方式通过组合 steps 或自定义step 定制更适合的编排管线



**原则**

- 一切LLM调用全部依赖于 gochat/pkg/core/Client 接口


## 管线化

基于 gochat/pkg/pipeline 机制构建 ReAct 管线
