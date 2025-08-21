# **GenAI-Powered OpenShift Audit Query System**

## Product Requirements Document (PRD)

**Version:** 1.0  
**Date:** August 1, 2025  
**Document Owner:** Platform Engineering / Security Engineering  
**Status:** First Draft  
**Author:** Mriganka Paul \[Senior Principal Software Engineer\]  
**Change Log:**  
	 Aug 17 \- Updated Basic, Intermediate and advanced query samples.  
 Aug 18 \- Updated with enhanced JSON schema validation	  
---

## 1\. Executive Summary

The GenAI-Powered OpenShift Audit Query System transforms complex OpenShift audit operations into natural language conversations, enabling cluster administrators and security teams to perform sophisticated audit queries without deep command-line expertise. By leveraging Generative AI and Model Context Protocol (MCP) servers, this system democratizes security monitoring and accelerates incident response across OpenShift environments.

### Key Value Propositions

- **Natural Language Audit Queries**: "Show me all CRD deletions by human users" instead of complex oc commands  
- **Intelligent Result Analysis**: AI-powered interpretation and threat correlation  
- **Security Context Awareness**: Deep understanding of OpenShift security patterns and RBAC  
- **Instant Expertise**: Convert junior admins into effective audit investigators  
- **Rapid Investigation**: Accelerate security incident response and troubleshooting

---

## 2\. Background

### 2.1 What is OpenShift Audit Logging?

OpenShift Container Platform auditing provides a security-relevant chronological set of records documenting the sequence of activities that have affected the system by individual users, administrators, or other components of the system. Audit works at the API server level, logging all requests coming to the server.

OpenShift audit logs track a wide range of actions performed by users and services. These logs capture data such as:

- Who accessed the system  
- What actions were taken  
- When these actions occurred

Each audit log entry contains detailed information including:

- User information  
- Request URI  
- HTTP verb  
- Source IPs  
- Timestamps  
- Authorization decisions

### 2.2 Architecture

OpenShift audit logging operates at multiple API server levels with these key components:

#### 2.2.1 API Servers That Generate Audit Logs

You can view the logs for the OpenShift API server, Kubernetes API server, OpenShift OAuth API server, and OpenShift OAuth server for each control plane node.

- **OpenShift API Server** \- Handles OpenShift-specific resources  
- **Kubernetes API Server** \- Handles core Kubernetes resources  
- **OpenShift OAuth API Server** \- Handles authentication API requests  
- **OpenShift OAuth Server** \- Handles OAuth login flows

#### 2.2.2 Log Storage

Audit logs are stored locally on each control plane node. Files are located in specific paths:

- `/openshift-apiserver/` \- OpenShift API server audit logs  
- `/kube-apiserver/` \- Kubernetes API server audit logs  
- `/oauth-apiserver/` \- OAuth API server audit logs  
- `/oauth-server/` \- OAuth server audit logs  
- `/var/log/audit/audit.log` \- Node audit system (auditd) logs

**Note**: The audit logs from the Kubernetes apiserver and the OpenShift apiserver are also included in the node audit system logs.

### 2.3 Extensible Components

While the core audit logging architecture is fixed, there are several extensible aspects:

#### 2.3.1 Log Forwarding

The Red Hat OpenShift Logging Operator provides the following cluster roles:

- **collect-audit-logs** \- Enables collection of audit logs  
- **collect-application-logs** \- Enables collection of application logs  
- **collect-infrastructure-logs** \- Enables collection of infrastructure logs

These roles enable the collector to collect different types of logs respectively.

You can forward audit logs to external systems using:

- **Elasticsearch**  
- **Kafka**  
- **AWS CloudWatch**  
- **Azure Monitor**  
- **Google Cloud Logging**  
- **Generic HTTP endpoints**  
- **Syslog protocols**

#### 2.3.2 Custom Rules and Profiles

The audit policy system allows for extensible configuration through custom rules and profiles.

### 2.4 Controls Available When Setting Up Auditing

#### 2.4.1 Audit Policy Profiles

OpenShift Container Platform provides the following predefined audit policy profiles:

- **Default** \- By default, OpenShift Container Platform uses the Default audit log profile (logs metadata only)  
- **WriteRequestBodies** \- Logs metadata and request bodies  
- **AllRequestBodies** \- Logs metadata, request bodies, and response bodies  
- **None** \- Disables audit logging (not recommended)

#### 2.4.2 Custom Rules

You can configure an audit log policy that defines custom rules. You can specify multiple groups and define which profile to use for that group. These custom rules take precedence over the top-level profile field.

#### 2.4.3 Group-Based Configuration

You can apply different audit levels to different user groups (e.g., `system:authenticated:oauth`, `system:authenticated`).

### 2.5 How to Set Up Audit Logging

#### 2.5.1 Prerequisites

You have access to the cluster as a user with the cluster-admin role.

#### 2.5.2 Basic Setup (Change Audit Profile)

1. **Edit the APIServer configuration:**

`apiVersion: config.openshift.io/v1`  
`kind: APIServer`  
`metadata:`  
  `name: cluster`  
`spec:`  
  `audit:`  
    `profile: WriteRequestBodies  # or AllRequestBodies`

2. **Apply the changes:**

`oc edit apiserver cluster`

3. **Verify the rollout:**

Verify that a new revision of the Kubernetes API server pods is rolled out. It can take several minutes for all nodes to update to the new revision.

#### 2.5.3 Advanced Setup with Custom Rules

`apiVersion: config.openshift.io/v1`  
`kind: APIServer`  
`metadata:`  
  `name: cluster`  
`spec:`  
  `audit:`  
    `customRules:`  
    `- group: system:authenticated:oauth`  
      `profile: WriteRequestBodies`  
    `- group: system:authenticated`  
      `profile: AllRequestBodies`  
    `profile: Default  # fallback for unmatched groups`

### 2.6 Accessing Audit Logs

To view the audit logs, you can use the following commands:

#### 2.6.1 List Available Audit Logs

`# List available audit logs`  
`oc adm node-logs --role=master --path=openshift-apiserver/`

#### 2.6.2 View Specific Audit Log

`# View specific audit log`  
`oc adm node-logs <node_name> --path=openshift-apiserver/<log_name>`

#### 2.6.3 Filter Logs with jq

`# Filter logs by username`  
`oc adm node-logs node-1.example.com \`  
  `--path=openshift-apiserver/audit.log \`  
  `| jq 'select(.user.username == "myusername")'`

### 2.7 Collecting Audit Logs for Support

You can use the must-gather tool to collect the audit logs for debugging your cluster:

`oc adm must-gather -- /usr/bin/gather_audit_logs`

---

## 3\. Product Overview

### 3.1 Solution Description

The system enables administrators to perform OpenShift audit investigations using natural language, such as:

**Simple Queries:**

- "Who deleted the DataScienceCluster CRD yesterday?"  
- "Show me all failed authentication attempts in the last hour"  
- "List all admin actions by user john.doe this week"

**Complex Investigations:**

- "Find unusual API access patterns that don't match normal service account behavior"  
- "Show me privilege escalation attempts correlating with pod creation failures"  
- "Identify potential insider threats based on after-hours admin activity"

The system processes these requests using GenAI, translates them to appropriate oc commands via MCP servers, executes the queries safely, and provides intelligent analysis of results.

### 3.2 Target Users

- **Primary**: OpenShift cluster administrators and platform engineers  
- **Secondary**: Security operations teams and DevOps engineers  
- **Tertiary**: Development team leads requiring security visibility

---

## 4\. Goals and Success Metrics

### 4.1 Primary Goals

- **Democratize Security Monitoring**: Enable non-experts to perform complex audit queries  
- **Accelerate Incident Response**: Reduce investigation time from hours to minutes  
- **Enhance Operational Efficiency**: Automate repetitive audit query generation  
- **Reduce Skill Barrier**: Eliminate need for complex oc command expertise  
- **Enhance Security Posture**: Increase frequency and quality of security audits

### 4.2 Key Performance Indicators (KPIs)

- **Query Success Rate**: \>95% of natural language queries produce valid oc commands  
- **Time Reduction**: 90% reduction in audit query creation time  
- **Accuracy**: \>98% accuracy in command generation with safety validation  
- **User Adoption**: 80% of cluster admins actively using the system within 9 months  
- **Security Coverage**: 100% of OpenShift audit log sources accessible

### 4.3 Phases

- Initial Phase of the project will focus on querying on active logs.  
- Later phases will include smart file log discovery to handle log rotation.

### 4.4 Out of scope

- For the first version of the product it will be able access the logs stored in a single Openshift cluster and will not access external systems like cloud storage, log management platforms, Message Queue and streaming, Traditional systems such as sys log servers and SIEM platforms.


---

 

## 5\. User Stories and Use Cases

### 5.1 Core User Stories

Epic 1: Natural Language Query Translation

- US-001: As a cluster admin, I want to ask "who deleted the customer CRD?" so that I can quickly identify the responsible user without learning complex grep syntax  
- US-002: As a security analyst, I want to query "show me all privilege escalation attempts" so that I can investigate potential security incidents using plain English  
- US-003: As a platform engineer, I want to ask "what API calls failed with permission denied errors?" so that I can troubleshoot RBAC issues conversationally

Epic 2: Safe Execution and Validation

- US-007: As a cluster admin, I want query preview and validation so that I understand what commands will be executed before running them  
- US-008: As a platform engineer, I want read-only safety guarantees so that audit queries cannot accidentally modify cluster state  
- US-009: As a security team lead, I want audit trails of all queries so that I can track who investigated what and when

Epic 3: Intelligent Analysis and Correlation

- US-004: As an incident responder, I want the system to automatically correlate suspicious activities so that I can understand the full scope of security events  
- US-005: As a security analyst, I want to generate detailed audit reports by asking "show me all admin actions in the production namespace last week" so that I can analyze security patterns  
- US-006: As a security manager, I want baseline-aware anomaly detection so that the system highlights truly unusual audit patterns

---

## 6\. Functional Requirements

### 6.1 Natural Language Processing Engine

| Requirement ID | Feature | Description | Priority | Comments | Is MCP required? |
| :---- | :---- | :---- | :---- | :---- | :---- |
| FR-001 | LLM Query Parser | Use opensource LLM (Llama 3.1/Mistral) with system prompts to parse natural language into structured JSON parameters with 95% accuracy | P0 | **What it does**: Uses an open-source language model (like Llama 3.1 or Mistral) to convert natural language  queries into structured JSON parameters.  **Example**: "Who deleted the customer CRD yesterday?" becomes `{"log_source": "kube-apiserver", "patterns": ["customresourcedefinition", "delete", "customer"], "timeframe": "yesterday", "exclude": ["system:"]}` **Why it's critica`l`**: This is the core AI functionality that makes the system accessible to non-technical users  | **No, Pure LLM functionality with prompts** |
| FR-002 | Context Understanding | Understand OpenShift-specific entities through prompt engineering with examples | P0 | **What it does**: Understands OpenShift-specific terminology, resources, and relationships through carefully crafted prompts **Example**: Knows that "CRD" means CustomResourceDefinition, understands namespace hierarchy, recognizes service account patterns **Why it's critical**: Generic LLMs don't understand OpenShift concepts without this specialized knowledge | **No, LLM with OpenShift-specific prompts** |
| FR-003 | Query Validation | Validate generated parameters for safety and feasibility using rule-based checking | P0 | **What it does**: Checks that generated parameters are safe, reasonable, and won't cause system harm **Example**: Ensures timeframes aren't too broad (preventing system overload), validates resource names exist, prevents command injection **Why it's critical**: Prevents the AI from generating dangerous or nonsensical queries | **No, Rule-based validation in the Safety Validator component** |
| FR-004 | Intent Classification | Classify query intent (investigation, troubleshooting, monitoring) through prompt patterns | P1 | **What it does**: Categorizes queries by purpose (investigation, troubleshooting, monitoring) to optimize response handling **Example**: "Who deleted..." \= investigation, "Why is X failing..." \= troubleshooting, "Show me daily..." \= monitoring **Why it's useful**: Allows the system to provide context-appropriate responses and suggestions | **No, LLM prompt-based classification** |
| FR-005 | Multi-turn Conversations | Support follow-up questions using simple session state management | P1 | **What it does**: Maintains conversation context to handle follow-up questions and references **Example**: After asking "Who deleted the customer CRD?", user can ask "When did he do it?" and system remembers the user and resource **Why it's useful**: Makes investigations more natural and efficient  | **No Context Manager handles session state** |

### 6.2 Command Generation and Execution

| Requirement ID | Feature | Description | Priority | Comments | Is MCP Required? |
| :---- | :---- | :---- | :---- | :---- | :---- |
| FR-006 | oc Command Generation | Generate accurate oc node-logs commands for audit queries | P0 | **What it does**: Converts the structured JSON parameters into actual `oc adm node-logs` commands for querying audit logs **Example**: JSON parameters become `oc adm node-logs --role=master --path=kube-apiserver/audit.log | grep -i "customresourcedefinition" | grep -i "delete"`  **Why it's critical**: This is the bridge between AI understanding and actual OpenShift audit querying | **Yes MCP tool like**: generate\_audit\_query |
| FR-007 | Safety Validation | Ensure all generated commands are read-only and safe | P0 | **What it does**: Guarantees all generated commands are read-only and cannot modify cluster state **Example**: Only allows commands like `node-logs`, `get`, `describe` \- never `create`, `delete`, `patch` **Why it's critical**: Prevents accidental cluster modifications during audit investigations | **Yes,** Built into the MCP servers command generation with whitelisting |
| FR-008 | Query Preview | Show generated commands before execution with explanation | P0 | **What it does**: Shows users the actual command that will be executed before running it, with plain English explanation **Example**: Shows "This will search API server audit logs for CRD deletions by non-system users in the last 24 hours" **Why it's critical**: Builds user trust and allows verification before execution | **Yes.** The mcp server generates command previews and explanations |
| FR-009 | Historical Data Access | Access audit logs across configurable time ranges | P0 | **What it does**: Provides access to audit logs across configurable time ranges (hours, days, weeks) **Example**: Can query "yesterday", "last week", "between 2 PM and 4 PM on Tuesday" **Why it's critical**: Security investigations often require historical context | **Yes.** MCP server handles the actual command oc adm node-logs execution |
| FR-010 | Result Caching | Cache frequent queries for improved performance | P1 | **What it does**: Stores results of frequent queries to improve response times **Example**: Caches results for common queries like "daily authentication failures" for 15 minutes **Why it's useful**: Reduces load on the cluster and improves user experience |  |

### 6.3 Result Analysis and Intelligence

| Requirement ID | Feature | Description | Priority | Comments | Is MCP required? |
| :---- | :---- | :---- | :---- | :---- | :---- |
| FR-011 | Intelligent Summarization | Summarize audit results in business-friendly language | P0 | **What it does**: Converts raw audit log output into human-readable summaries and insights **Example**: Instead of raw JSON logs, provides "3 CRD deletions found: customer-service (john.doe, 2:30 PM), inventory (jane.smith, 3:15 PM)..." **Why it's critical**: Makes audit results accessible to non-technical users |  |
| FR-012 | Anomaly Detection | Highlight unusual patterns and potential security issues | P1 | **What it does**: Identifies unusual patterns in audit results that might indicate security issues **Example**: Flags "User typically active 9-5, but shows activity at 2 AM" or "Unusual spike in failed authentications" **Why it's useful**: Helps identify potential security threats automatically |  |
| FR-013 | Threat Correlation | Correlate events across time and services for investigation | P1 | **What it does**: Links related events across time and services to build investigation timelines **Example**: Correlates failed authentication attempts with subsequent privilege escalation attempts by the same user **Why it's useful**: Provides comprehensive security incident context  |  |
| FR-014 | Export Capabilities | Export results in multiple formats (JSON, CSV, PDF reports) | P0 | **What it does**: Allows saving and sharing results in multiple formats **Example**: Export investigation results as JSON for API integration, CSV for spreadsheets, PDF for reports **Why it's critical**: Enables integration with existing security workflows and compliance reporting | **The `parse_audit_results` MCP tool handles output formatting** |

### 6.4 Security and Access Control

| Requirement ID | Feature | Description | Priority | Comments | Is MCP required? |
| :---- | :---- | :---- | :---- | :---- | :---- |
| FR-015 | User Authentication | Authenticate users before allowing audit queries | P0 | **What it does**: Verifies user identity before allowing access to audit queries **Example**: Integrates with existing OpenShift authentication (OAuth, LDAP, etc.) **Why it's critical**: who gains access to the developed system | **No** API Gateway handles auth |
| FR-016 | Query Audit Trail | Log all queries, users, and results for accountability | P0 | **What it does**: Logs every query, who ran it, when, and what results were returned **Example**: Records "john.doe ran 'who deleted CRD' at 2025-01-29 14:30 UTC, returned 3 results" **Why it's critical**: Provides accountability and helps track investigation activities | **No** Logging service handles audit trails |
| FR-017 | Data Privacy | Ensure sensitive data in audit logs is handled securely | P0 | **What it does**: Ensures sensitive information in audit logs is protected during processing and storage **Example**: Masks or encrypts sensitive fields, implements data retention policies **Why it's critical**: Compliance with data protection regulations and security policies | **No** API Gateway and database handle privacy |
| FR-018 | Session Management | Secure session handling with timeout and authentication | P0 | **What it does**: Securely handles user sessions with appropriate timeouts and security controls **Example**: Sessions expire after inactivity, secure token handling, session invalidation **Why it's critical**: Prevents unauthorized access through session hijacking  | **No** API Gateway manages sessions |
| FR-019 | Permission Inheritance | Use existing oc client permissions for audit access | P0 | **What it does**: Uses the user's existing OpenShift permissions to determine what audit data they can access **Example**: If user can't access namespace "production", they can't query its audit logs either **Why it's critical**: Maintains existing security boundaries and principle of least privilege | **No** API Gateway enforces RBAC |

---

## 7\. MCP server’s specific roles

The **Audit Query MCP Server** acts as a **specialized connector** that:

1. **Receives** structured JSON from the LLM (after natural language processing)  
2. **Converts** JSON to safe `oc` commands using templates  
3. **Executes** commands with safety controls and timeouts  
4. **Returns** structured results back to the LLM for analysis

\# The MCP server only implements these 3 core tools:

@mcp\_tool

def generate\_audit\_query(structured\_params: dict) \-\> dict:

    """FR-006, FR-007, FR-008: Generate safe oc commands"""

@mcp\_tool  

def execute\_audit\_query(command: str) \-\> str:

    """FR-009: Execute oc node-logs commands safely"""

@mcp\_tool

def parse\_audit\_results(raw\_output: str, query\_context: dict) \-\> dict:

    """FR-014: Format results for LLM consumption"""

### 7.1  Why this architecture make sense

**LLM handles intelligence** (understanding, analysis, conversation)  
**MCP server handles execution** (safe command generation and running)  
**API Gateway handles security** (auth, sessions, permissions)

This separation means the MCP server has a **narrow, focused responsibility** \- it's essentially a "safe OpenShift command executor" rather than trying to do AI processing or security management.

### 7.2  Business Justification

1. **Risk Mitigation**: Audit queries can be dangerous if misconfigured. MCP's structured validation prevents security incidents.  
2. **ROI**: The upfront complexity pays off through multi-platform AI support and reduced integration maintenance.  
3. **Compliance**: The comprehensive audit trails and testing framework meet enterprise security requirements.  
4. **Future Value**: As AI adoption grows, the standardized MCP interface positions the team ahead of custom integration approaches.

The business question isn't "**Could we build this simpler?**" but "**What's the cost of a security incident from poorly validated audit commands?**"  
   
---

## 8\. Technical Architecture

### 8.1 High-Level System Design

System Architecture Overview

| Layer | Component | Description |
| :---- | :---- | :---- |
| User Interface Layer | CLI Tool | Command-line interface for natural language audit queries |
| API Gateway & Authentication | Auth Service | User authentication and session management |
|  | Session Management | Secure session handling with timeout controls |
| GenAI Processing Layer | Audit Query Agent | Core AI processing with three components: |
|  | LLM Engine (Llama/Mistral) | Natural language to structured JSON conversion |
|  | Context Manager (Session State) | Conversation context and pronoun resolution |
|  | Safety Validator (Rule Checker) | Command safety and input validation |
| Audit Query MCP Server | Query Generator | Convert JSON parameters to oc commands |
|  | Executor | Safe command execution with timeout protection |
|  | Parser | Result parsing and structured output generation |
| OpenShift Cluster | API Server Audit Logs | Kubernetes API audit trail |
|  | OAuth Server Audit Logs | Authentication and authorization logs |
|  | Node auditd Audit Logs | Node-level system audit logs |

Data Flow:

1. User submits natural language query via CLI  
2. API Gateway authenticates user and manages session  
3. Audit Query Agent processes query through LLM Engine  
4. Context Manager maintains conversation state  
5. Safety Validator ensures query safety  
6. MCP Server generates and executes oc commands  
7. Results are parsed and returned to user  
8. All queries and results are logged for audit trail  
   

### 8.2 Core Components

| Component | Technology | Justification |
| :---- | :---- | :---- |
| LLM Engine | Llama 3.1 8B / Mistral 7B via vLLM (any open source model that performs) | Opensource models with strong instruction following for audit query conversion |
| Context Manager | Python session state management | Simple conversation context tracking and pronoun resolution |
| Safety Validator | Rule-based validation logic | Whitelist approach for command safety and input sanitization |
| Audit Query MCP Server | Custom Go service implementing MCP protocol | Specialized connector for safe audit command generation and execution |
| API Gateway | Istio Gateway | Authentication, rate limiting, and request routing |
| Query Cache | Redis | Performance optimization for repeated audit queries |
| Audit Database | PostgreSQL | Persistent storage for query history and results |

### 

### 6.3 Audit Query Agent Architecture

Core Responsibility: Process natural language audit questions through a simple three-component pipeline

Component Details:

LLM Engine (The AI Component):

- Technology: Opensource models (Llama 3.1 8B, Mistral 7B) served via vLLM  
- Function: Convert natural language → structured JSON parameters using prompt engineering  
- Input: "Who deleted the customer CRD yesterday?"  
- Output: {"log\_source": "kube-apiserver", "patterns": \["customresourcedefinition", "delete", "customer"\], "timeframe": "yesterday", "exclude": \["system:"\]}  
- Implementation: System prompt with few-shot examples, no model training required

Context Manager (Simple Python State):

- Technology: Basic Python session management  
- Function: Track conversation history and resolve references  
- Capabilities: Remember last user/resource/timeframe, handle "he", "that user", "around that time"  
- Implementation: Dictionary-based state storage with simple substitution logic

Safety Validator (Rule-Based Checker):

- Technology: Rule-based validation logic  
- Function: Ensure generated queries are safe and reasonable  
- Checks: Command whitelist, input sanitization, timeframe validation, injection prevention  
- Implementation: Regex patterns and predefined rule sets

### 6.4 Audit Query MCP Server

Core Responsibility: Convert validated JSON parameters into safe oc commands, execute them, and return structured results.

Essential Tools:

```py
@mcp_tool
def generate_audit_query(structured_params: dict) -> dict:
    """Convert JSON parameters to oc audit command with safety validation"""

@mcp_tool  
def execute_audit_query(command: str) -> str:
    """Safely execute the oc command and return raw results"""

@mcp_tool
def parse_audit_results(raw_output: str, query_context: dict) -> dict:
    """Parse oc output into structured, readable format"""
```

Safety Features:

- Read-only Enforcement: All commands strictly read-only (node-logs, get, describe)  
- Input Sanitization: Prevent command injection and malicious inputs  
- Timeout Protection: Kill long-running queries automatically  
- Error Handling: Convert oc failures into user-friendly messages

## 10\. Risk Assessment and Mitigation

### Technical Risks

| Risk | Impact | Probability | Mitigation Strategy |
| :---- | :---- | :---- | :---- |
| GenAI hallucination creating unsafe commands | High | Medium | Multi-layer validation, command whitelisting, read-only enforcement |
| Performance degradation with large audit datasets | Medium | High | Query optimization, result caching, timeout controls |
| Security vulnerabilities in command generation | High | Low | Extensive security testing, command templates, audit logging |
| Limited query pattern coverage | Medium | Medium | Iterative pattern library expansion, user feedback integration |
| Availability of audit data | Medium | Medium | For complex and correlated queries, additional synthetic data needs to be created. |

### Business Risks

| Risk | Impact | Probability | Mitigation Strategy |
| :---- | :---- | :---- | :---- |
| Low user adoption due to trust issues | High | Medium | Extensive testing, transparency in command generation, gradual rollout |
| Performance concerns with large-scale deployment | Medium | Low | Comprehensive load testing, scalable architecture design |

## 11\. Conclusion

The GenAI-Powered OpenShift Audit Query System represents a focused, practical approach to simplifying security monitoring in OpenShift environments. By concentrating on the core problem of audit query complexity, this system will significantly reduce the barrier to effective security monitoring while maintaining safety and reliability.

The streamlined architecture, centered around a single specialized Audit Query MCP Server, provides a clear path to implementation with manageable complexity and risk. The phased approach ensures steady progress toward a production-ready system that delivers immediate value to cluster administrators and security teams.

## 12\. Appendices 

## Sample Queries

1. Cross platform Date validation \- [https://github.com/mpaulgreen/audit-queries-functional/blob/main/cross\_platform\_date.sh](https://github.com/mpaulgreen/audit-queries-functional/blob/main/cross_platform_date.sh)  
2. Basic Queries \- [https://github.com/mpaulgreen/audit-queries-functional/blob/main/basic\_queries.md](https://github.com/mpaulgreen/audit-queries-functional/blob/main/basic_queries.md)  
3. Intermediate Queries \- [https://github.com/mpaulgreen/audit-queries-functional/blob/main/intermediate\_queries.md](https://github.com/mpaulgreen/audit-queries-functional/blob/main/intermediate_queries.md)  
4. Advanced Queries \- [https://github.com/mpaulgreen/audit-queries-functional/blob/main/advanced\_queries.md](https://github.com/mpaulgreen/audit-queries-functional/blob/main/advanced_queries.md) 

## Enhanced JSON Schema

- JSON schema analysis [https://github.com/mpaulgreen/audit-queries-functional/blob/main/json\_schema\_analysis.md](https://github.com/mpaulgreen/audit-queries-functional/blob/main/json_schema_analysis.md)   
- Enhanced JSON schema [https://github.com/mpaulgreen/audit-queries-functional/blob/main/enhanced\_json\_schema.md](https://github.com/mpaulgreen/audit-queries-functional/blob/main/enhanced_json_schema.md)   
- Schema Validation Rule [https://github.com/mpaulgreen/audit-queries-functional/blob/main/schema\_validation\_rules.md](https://github.com/mpaulgreen/audit-queries-functional/blob/main/schema_validation_rules.md) 