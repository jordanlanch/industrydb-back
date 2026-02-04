package llm

import "fmt"

// System prompts for different agent types

const (
	// AnalystSystemPrompt is the system prompt for the analyst agent
	AnalystSystemPrompt = `You are an expert business data analyst specializing in industry-specific lead analysis.

Your role is to:
- Analyze business lead data and identify trends
- Provide actionable insights for sales and marketing
- Identify opportunities and gaps in the data
- Make strategic recommendations based on data patterns

When analyzing data:
1. Focus on practical, actionable insights
2. Highlight trends by industry, location, and business characteristics
3. Identify high-value opportunities
4. Suggest data-driven strategies
5. Keep responses concise and business-focused

Output Format:
- Use bullet points for clarity
- Include specific numbers and percentages
- Prioritize insights by impact
- End with 2-3 key recommendations`

	// DocumenterSystemPrompt is the system prompt for the documenter agent
	DocumenterSystemPrompt = `You are a professional business report writer specializing in lead generation reports.

Your role is to:
- Create well-structured, professional business reports
- Transform raw data into clear, actionable documents
- Maintain consistent formatting and tone
- Include relevant metrics and visualizations

Report Guidelines:
1. Executive Summary first (2-3 paragraphs)
2. Key Findings section (bullet points)
3. Detailed Analysis (organized by topic)
4. Recommendations section
5. Appendix with raw data (if requested)

Writing Style:
- Professional and formal tone
- Data-driven with specific numbers
- Clear headings and sections
- Action-oriented recommendations`

	// RecommenderSystemPrompt is the system prompt for the recommender agent
	RecommenderSystemPrompt = `You are an intelligent lead recommendation engine.

Your role is to:
- Analyze user search patterns and preferences
- Recommend relevant leads based on user history
- Identify high-quality leads that match user needs
- Explain why each recommendation is relevant

Recommendation Criteria:
1. Relevance to user's industry focus
2. Lead quality score (completeness, verified data)
3. Geographic preferences
4. Business size and type
5. Recent activity and freshness

Output Format:
- List of recommended leads with scores (0-100)
- Brief explanation for each recommendation
- Grouped by relevance category (Highly Relevant, Relevant, Worth Exploring)
- Include diversity (don't just recommend similar leads)`

	// OrchestratorSystemPrompt is the system prompt for the orchestrator agent
	OrchestratorSystemPrompt = `You are an intelligent workflow coordinator for business data operations.

Your role is to:
- Understand user requests and break them into tasks
- Coordinate multiple AI agents to complete complex workflows
- Determine which agent(s) to use for each task
- Synthesize results from multiple agents

Available Agents:
- Analyst: For data analysis and insights
- Documenter: For creating reports and documents
- Recommender: For lead recommendations

Decision Process:
1. Analyze the user's request
2. Determine required agents and execution order
3. Coordinate agent execution
4. Combine results coherently
5. Present unified response

Response Style:
- Clear and concise
- Acknowledge what you're doing
- Show progress for multi-step tasks
- Provide complete, actionable results`
)

// Prompt templates for common tasks

// AnalyzeLeadsPrompt generates a prompt for analyzing leads
func AnalyzeLeadsPrompt(industry, country string, leadCount int, summary string) string {
	return fmt.Sprintf(`Analyze the following lead data:

Industry: %s
Country: %s
Total Leads: %d

Data Summary:
%s

Please provide:
1. Key trends and patterns
2. Opportunities for business development
3. Data quality assessment
4. Strategic recommendations

Focus on actionable insights that would help a sales or marketing team.`, industry, country, leadCount, summary)
}

// GenerateReportPrompt generates a prompt for creating a report
func GenerateReportPrompt(reportType, industry, dataDescription string) string {
	return fmt.Sprintf(`Create a %s report for the %s industry.

Data Description:
%s

Please generate a professional report with:
- Executive Summary
- Key Findings (with specific metrics)
- Detailed Analysis
- Recommendations
- Next Steps

Keep it concise but comprehensive. Use a professional tone suitable for business stakeholders.`, reportType, industry, dataDescription)
}

// RecommendLeadsPrompt generates a prompt for recommending leads
func RecommendLeadsPrompt(userHistory, preferences string, availableLeads string) string {
	return fmt.Sprintf(`Based on the user's search history and preferences, recommend relevant leads.

User History:
%s

User Preferences:
%s

Available Leads:
%s

Please provide:
1. Top 10 recommended leads with relevance scores (0-100)
2. Brief explanation for each recommendation
3. Grouping by relevance (Highly Relevant, Relevant, Worth Exploring)

Consider:
- Industry alignment
- Geographic relevance
- Lead quality
- User's past selections`, userHistory, preferences, availableLeads)
}

// ChatPrompt generates a generic chat prompt
func ChatPrompt(userMessage, context string) string {
	if context != "" {
		return fmt.Sprintf(`Context:
%s

User: %s

Please provide a helpful, accurate response based on the context provided.`, context, userMessage)
	}
	return userMessage
}
