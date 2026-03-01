package leetcode

import "time"

const leetcodeGraphQL = "https://leetcode.com/graphql"

const matchedUserQuery = `
  query userPublicProfile($username: String!) {
    matchedUser(username: $username) {
      profile { ranking userAvatar realName }
    }
  }
`

const questionsSubmittedQuery = `
  query recentAcSubmissions($username: String!, $limit: Int!) {
    recentAcSubmissionList(username: $username, limit: $limit) {
      titleSlug
      timestamp
    }
  }
`

const languageProblemCountQuery = `
  query languageStats($username: String!) {
    matchedUser(username: $username) {
      languageProblemCount { languageName problemsSolved }
    }
  }
`

const allQuestionListQuery = `
query problemsetQuestionList($categorySlug: String, $limit: Int, $skip: Int, $filters: QuestionListFilterInput) {
  problemsetQuestionList: questionList(
    categorySlug: $categorySlug
    limit: $limit
    skip: $skip
    filters: $filters
  ) {
    questions: data {
      frontendQuestionId: questionFrontendId
      titleSlug
    }
  }
}
`

var qCache qCacheState

const qCacheTTL = time.Hour
