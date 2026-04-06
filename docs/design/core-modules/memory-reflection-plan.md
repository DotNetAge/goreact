# 反思与计划存储

> **相关文档**: [Memory 模块概述](memory-module.md) | [节点类型定义](memory-nodes.md) | [接口设计](memory-interfaces.md)

反思（Reflection）与计划（Plan）是 Memory 演进范式的核心组件，使系统能够从失败中学习，并在后续任务中复用成功经验。通过 `Reflections()`、`Plans()`、`Trajectories()` 访问器管理相关节点。

## 1. 访问器概述

| 访问器         | 节点类型   | 说明         |
| -------------- | ---------- | ------------ |
| Reflections()  | Reflection | 反思节点     |
| Plans()        | Plan       | 计划节点     |
| Trajectories() | Trajectory | 执行轨迹节点 |

```go
reflections := memory.Reflections()
plans := memory.Plans()
trajectories := memory.Trajectories()
```

## 2. 反思机制概述

反思机制通过分析失败的任务执行轨迹，提取经验教训，生成指导性建议：

```mermaid
flowchart TB
    subgraph 失败任务
        Session[失败会话]
        Trajectory[执行轨迹]
    end
    
    subgraph 反思分析
        Session --> Analyze[分析失败原因]
        Trajectory --> Analyze
        Analyze --> Extract[提取经验教训]
        Extract --> Generate[生成反思建议]
    end
    
    subgraph 存储与复用
        Generate --> Store[存储Reflection节点]
        Store --> Index[索引到Memory]
        Index --> Retrieve[后续任务检索]
    end
```

## 2. Reflection 节点设计

```mermaid
classDiagram
    class Reflection {
        +Name string
        +SessionName string
        +TrajectoryName string
        +FailureReason string
        +Analysis string
        +Heuristic string
        +Suggestions []string
        +Score float64
        +TaskType string
        +CreatedAt time.Time
    }
```

**字段说明**：

| 字段           | 说明                               |
| -------------- | ---------------------------------- |
| Name           | 反思唯一标识                       |
| SessionName    | 关联的会话标识                     |
| TrajectoryName | 关联的执行轨迹标识                 |
| FailureReason  | 失败原因摘要                       |
| Analysis       | 详细的失败分析                     |
| Heuristic      | 启发式建议（用于指导下一次尝试）   |
| Suggestions    | 具体改进建议列表                   |
| Score          | 反思质量分数（用于过滤低质量反思） |
| TaskType       | 任务类型（用于分类检索）           |
| CreatedAt      | 创建时间                           |

## 3. 反思生成流程

```mermaid
sequenceDiagram
    participant Reactor as 反应器
    participant Reflector as 反思器
    participant LLM as LLM
    participant Memory as Memory

    Reactor->>Reactor: 任务执行失败
    Reactor->>Reflector: 触发反思(sessionName)
    
    Reflector->>Memory: 获取执行轨迹
    Memory-->>Reflector: 返回Trajectory
    
    Reflector->>LLM: 分析失败原因
    LLM-->>Reflector: 返回分析结果
    
    Reflector->>LLM: 生成改进建议
    LLM-->>Reflector: 返回建议
    
    Reflector->>Reflector: 计算反思质量分数
    
    alt 分数 >= 阈值
        Reflector->>Memory: 存储Reflection节点
        Memory-->>Reflector: 确认存储
    else 分数 < 阈值
        Reflector->>Reflector: 丢弃低质量反思
    end
    
    Reflector-->>Reactor: 返回反思结果
```

### 3.1 反思质量评分

| 维度     | 权重 | 说明                         |
| -------- | ---- | ---------------------------- |
| 具体性   | 0.3  | 建议是否具体可操作           |
| 相关性   | 0.3  | 建议是否与失败原因直接相关   |
| 可执行性 | 0.2  | 建议是否可在后续任务中执行   |
| 新颖性   | 0.2  | 建议是否提供了新的视角或方法 |

**分数计算**：

```
Score = Specificity * 0.3 + Relevance * 0.3 + Executability * 0.2 + Novelty * 0.2
```

### 3.2 反思阈值配置

```go
type ReflectionConfig struct {
    MinScoreThreshold    float64   // 最低质量分数阈值，默认 0.6
    MaxReflectionsPerDay int       // 每日最大反思数，默认 100
    RetentionDays        int       // 反思保留天数，默认 30
    EnableAutoReflection bool      // 是否启用自动反思，默认 true
}
```

## 4. 反思检索与应用

```mermaid
sequenceDiagram
    participant Thinker as 思考器
    participant Memory as Memory
    participant GraphRAG as GraphRAG

    Thinker->>Memory: 检索相关反思(taskType, query)
    Memory->>GraphRAG: 语义检索Reflection节点
    GraphRAG-->>Memory: 返回相关反思列表
    
    Memory->>Memory: 按分数和时间排序
    Memory->>Memory: 过滤过期反思
    Memory-->>Thinker: 返回Top-K反思
    
    Thinker->>Thinker: 将反思注入Prompt
    Thinker->>Thinker: 基于反思调整策略
```

### 4.1 反思检索策略

| 策略     | 说明                       |
| -------- | -------------------------- |
| 任务类型 | 优先检索相同类型的任务反思 |
| 语义相似 | 检索语义相似的失败场景     |
| 时间衰减 | 最近的反思权重更高         |
| 质量过滤 | 只返回分数超过阈值的反思   |

### 4.2 反思注入模板

```markdown
{{if .Reflections}}
<reflections>
以下是相关的历史经验教训，请参考：

{{range .Reflections}}
**失败原因**: {{.FailureReason}}
**分析**: {{.Analysis}}
**建议**: {{.Heuristic}}
{{end}}
</reflections>
{{end}}
```

## 5. 计划存储机制

计划（Plan）用于存储任务执行计划，支持计划复用和相似任务优化：

```mermaid
classDiagram
    class Plan {
        +Name string
        +SessionName string
        +Goal string
        +Steps []PlanStep
        +Status PlanStatus
        +Success bool
        +TaskType string
        +CreatedAt time.Time
        +CompletedAt time.Time
    }
    
    class PlanStep {
        +Index int
        +Description string
        +ExpectedAction string
        +ExpectedOutcome string
        +Status StepStatus
        +ActualOutcome string
        +Deviations []string
    }
    
    class PlanStatus {
        <<enumeration>>
        Pending
        InProgress
        Completed
        Failed
        Replanned
    }
    
    class StepStatus {
        <<enumeration>>
        Pending
        InProgress
        Completed
        Failed
        Skipped
    }
    
    Plan "1" *-- "*" PlanStep
    Plan --> PlanStatus
    PlanStep --> StepStatus
```

### 5.1 Plan 字段说明

| 字段        | 说明                     |
| ----------- | ------------------------ |
| Name        | 计划唯一标识             |
| SessionName | 关联的会话标识           |
| Goal        | 计划目标                 |
| Steps       | 计划步骤列表             |
| Status      | 计划状态                 |
| Success     | 计划是否成功完成         |
| TaskType    | 任务类型（用于分类检索） |
| CreatedAt   | 创建时间                 |
| CompletedAt | 完成时间                 |

### 5.2 PlanStep 字段说明

| 字段            | 说明                     |
| --------------- | ------------------------ |
| Index           | 步骤索引                 |
| Description     | 步骤描述                 |
| ExpectedAction  | 预期动作                 |
| ExpectedOutcome | 预期结果                 |
| Status          | 步骤状态                 |
| ActualOutcome   | 实际结果                 |
| Deviations      | 偏差记录（与预期的差异） |

## 6. 计划生成与执行流程

```mermaid
sequenceDiagram
    participant Thinker as 思考器
    participant Planner as 计划器
    participant Memory as Memory
    participant Reactor as 反应器

    Thinker->>Memory: 检索相似计划(goal)
    Memory-->>Thinker: 返回相似计划列表
    
    alt 存在相似计划
        Thinker->>Thinker: 复用并调整计划
    else 无相似计划
        Thinker->>Planner: 生成新计划
        Planner-->>Thinker: 返回计划
    end
    
    Thinker->>Memory: 存储Plan节点
    Memory-->>Thinker: 确认存储
    
    Thinker->>Reactor: 执行计划
    Reactor->>Reactor: 执行各步骤
    
    loop 每个步骤
        Reactor->>Memory: 更新步骤状态
    end
    
    Reactor-->>Thinker: 返回执行结果
    Thinker->>Memory: 更新计划最终状态
```

### 6.1 计划复用策略

```mermaid
flowchart TB
    Goal[任务目标] --> Retrieve[检索相似计划]
    Retrieve --> Similarity{相似度分析}
    
    Similarity --> |相似度 >= 0.8| Reuse[直接复用]
    Similarity --> |0.5 <= 相似度 < 0.8| Adapt[调整适配]
    Similarity --> |相似度 < 0.5| Generate[生成新计划]
    
    Reuse --> Validate[验证计划可行性]
    Adapt --> Validate
    Generate --> Validate
    
    Validate --> |可行| Execute[执行计划]
    Validate --> |不可行| Generate
```

### 6.2 相似度计算

```go
func CalculatePlanSimilarity(goal1, goal2 string, steps1, steps2 []PlanStep) float64 {
    // 目标语义相似度
    goalSimilarity := semanticSimilarity(goal1, goal2)
    
    // 步骤结构相似度
    stepSimilarity := calculateStepSimilarity(steps1, steps2)
    
    // 加权平均
    return goalSimilarity * 0.4 + stepSimilarity * 0.6
}
```

## 7. 执行轨迹存储

执行轨迹（Trajectory）记录任务执行的完整过程：

```mermaid
classDiagram
    class Trajectory {
        +Name string
        +SessionName string
        +Steps []TrajectoryStep
        +Thoughts []*Thought
        +Actions []*Action
        +Observations []*Observation
        +Success bool
        +FailurePoint int
        +FinalResult string
        +Duration time.Duration
        +Summary string
    }
    
    class TrajectoryStep {
        +Index int
        +Thought *Thought
        +Action *Action
        +Observation *Observation
        +Timestamp time.Time
    }
    
    class Thought {
        +Content string
        +Reasoning string
        +Decision string
        +Confidence float64
    }
    
    class Action {
        +Type string
        +Target string
        +Params map
        +Result string
    }
    
    class Observation {
        +Content string
        +Source string
        +Insights []string
    }
    
    Trajectory "1" *-- "*" TrajectoryStep
    TrajectoryStep --> Thought
    TrajectoryStep --> Action
    TrajectoryStep --> Observation
```

### 7.1 Trajectory 字段说明

| 字段         | 说明                   |
| ------------ | ---------------------- |
| Name         | 轨迹唯一标识           |
| SessionName  | 关联的会话标识         |
| Steps        | 执行步骤列表           |
| Thoughts     | 思考过程记录           |
| Actions      | 执行动作记录           |
| Observations | 观察结果记录           |
| Success      | 任务是否成功           |
| FailurePoint | 失败点索引（如果失败） |
| FinalResult  | 最终结果               |
| Duration     | 执行时长               |
| Summary      | 执行摘要               |

## 8. 反思-计划-轨迹关系图

```mermaid
graph TB
    subgraph 会话执行
        S[Session] --> |执行| T[Trajectory]
        S --> |关联| P[Plan]
    end
    
    subgraph 失败分析
        T --> |失败时| R[Reflection]
        R --> |基于| T
        R --> |学习来源| S
    end
    
    subgraph 计划复用
        P1[Plan 1] --> |相似| P2[Plan 2]
        R --> |指导| P2
    end
    
    subgraph 关系类型
        S --HAS_TRAJECTORY--> T
        S --HAS_PLAN--> P
        T --TRAJECTORY_OF--> S
        R --BASED_ON--> T
        R --LEARNED_FROM--> S
        R --INFORMS--> P2
        P1 --SIMILAR_TO--> P2
    end
```

## 9. 反思与计划服务接口

```mermaid
classDiagram
    class ReflectionService {
        <<interface>>
        +CreateReflection(ctx context.Context, sessionName string) (*Reflection, error)
        +GetReflection(ctx context.Context, reflectionName string) (*Reflection, error)
        +ListReflections(ctx context.Context, opts ...ReflectionOption) ([]*Reflection, error)
        +DeleteReflection(ctx context.Context, reflectionName string) error
        +GetRelevantReflections(ctx context.Context, taskType string, query string, topK int) ([]*Reflection, error)
    }
    
    class PlanService {
        <<interface>>
        +CreatePlan(ctx context.Context, sessionName string, goal string) (*Plan, error)
        +GetPlan(ctx context.Context, planName string) (*Plan, error)
        +UpdatePlan(ctx context.Context, plan *Plan) error
        +DeletePlan(ctx context.Context, planName string) error
        +FindSimilarPlans(ctx context.Context, goal string, threshold float64) ([]*Plan, error)
        +UpdatePlanStep(ctx context.Context, planName string, stepIndex int, status StepStatus, outcome string) error
    }
    
    class TrajectoryService {
        <<interface>>
        +CreateTrajectory(ctx context.Context, sessionName string) (*Trajectory, error)
        +AddStep(ctx context.Context, trajectoryName string, step *TrajectoryStep) error
        +GetTrajectory(ctx context.Context, trajectoryName string) (*Trajectory, error)
        +MarkComplete(ctx context.Context, trajectoryName string, success bool, result string) error
    }
```

**方法说明**：

| 服务              | 方法                   | 说明         |
| ----------------- | ---------------------- | ------------ |
| ReflectionService | CreateReflection       | 创建反思     |
| ReflectionService | GetReflection          | 获取反思     |
| ReflectionService | ListReflections        | 列出反思     |
| ReflectionService | DeleteReflection       | 删除反思     |
| ReflectionService | GetRelevantReflections | 获取相关反思 |
| PlanService       | CreatePlan             | 创建计划     |
| PlanService       | GetPlan                | 获取计划     |
| PlanService       | UpdatePlan             | 更新计划     |
| PlanService       | DeletePlan             | 删除计划     |
| PlanService       | FindSimilarPlans       | 查找相似计划 |
| PlanService       | UpdatePlanStep         | 更新计划步骤 |
| TrajectoryService | CreateTrajectory       | 创建轨迹     |
| TrajectoryService | AddStep                | 添加步骤     |
| TrajectoryService | GetTrajectory          | 获取轨迹     |
| TrajectoryService | MarkComplete           | 标记完成     |

## 10. 配置选项

```go
type ReflectionPlanConfig struct {
    // 反思配置
    ReflectionConfig ReflectionConfig
    
    // 计划配置
    PlanConfig PlanConfig
    
    // 轨迹配置
    TrajectoryConfig TrajectoryConfig
}

type PlanConfig struct {
    EnablePlanReuse       bool      // 是否启用计划复用，默认 true
    SimilarityThreshold   float64   // 相似度阈值，默认 0.7
    MaxPlanSteps          int       // 最大计划步骤数，默认 20
    PlanRetentionDays     int       // 计划保留天数，默认 90
}

type TrajectoryConfig struct {
    EnableTrajectoryStore bool      // 是否存储轨迹，默认 true
    MaxTrajectorySteps    int       // 最大轨迹步骤数，默认 100
    TrajectoryRetentionDays int     // 轨迹保留天数，默认 30
    EnableSummary         bool      // 是否生成摘要，默认 true
}
```
