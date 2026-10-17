import type { GraphQLPayload } from './types'
import { toRecord, requestFirstMatch } from './internal'

export async function executeGraphQL(query: string, variables?: Record<string, unknown>): Promise<GraphQLPayload> {
  return requestFirstMatch<GraphQLPayload>(
    ['/graphql'],
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: variables ? { query, variables } : { query },
    },
    (response, candidate) => ({
      body: toRecord(response.data),
      evidence: { label: 'GraphQL', path: candidate },
    }),
  )
}