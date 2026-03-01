import { useTranslation } from "react-i18next";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/Card";
import type { MatchScoreResponse } from "@/shared/types/api";

interface MatchScoreCardProps {
  data: MatchScoreResponse;
}

function getScoreColor(score: number): string {
  if (score >= 70) return "text-green-600";
  if (score >= 40) return "text-yellow-600";
  return "text-red-600";
}

function getBarColor(score: number): string {
  if (score >= 70) return "bg-green-500";
  if (score >= 40) return "bg-yellow-500";
  return "bg-red-500";
}

function getCircleColor(score: number): string {
  if (score >= 70) return "border-green-500 text-green-600";
  if (score >= 40) return "border-yellow-500 text-yellow-600";
  return "border-red-500 text-red-600";
}

export function MatchScoreCard({ data }: MatchScoreCardProps) {
  const { t } = useTranslation();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("applications.matchScore.overallScore")}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Overall Score Circle */}
        <div className="flex justify-center">
          <div
            className={`flex items-center justify-center w-24 h-24 rounded-full border-4 ${getCircleColor(data.overall_score)}`}
          >
            <span className="text-3xl font-bold">{data.overall_score}%</span>
          </div>
        </div>

        {/* Categories */}
        {data.categories.length > 0 && (
          <div>
            <h4 className="text-sm font-semibold mb-3">
              {t("applications.matchScore.categories")}
            </h4>
            <div className="space-y-3">
              {data.categories.map((category) => (
                <div key={category.name}>
                  <div className="flex justify-between text-sm mb-1">
                    <span>{category.name}</span>
                    <span className={getScoreColor(category.score)}>
                      {category.score}%
                    </span>
                  </div>
                  <div className="w-full bg-muted rounded-full h-2">
                    <div
                      className={`h-2 rounded-full transition-all ${getBarColor(category.score)}`}
                      style={{ width: `${category.score}%` }}
                    />
                  </div>
                  <p className="text-xs text-muted-foreground mt-1">
                    {category.details}
                  </p>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Strengths */}
        {data.strengths.length > 0 && (
          <div>
            <h4 className="text-sm font-semibold mb-2">
              {t("applications.matchScore.strengths")}
            </h4>
            <div className="flex flex-wrap gap-2">
              {data.strengths.map((strength) => (
                <span
                  key={strength}
                  className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800 dark:bg-green-900 dark:text-green-200"
                >
                  {strength}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Missing Keywords */}
        {data.missing_keywords.length > 0 && (
          <div>
            <h4 className="text-sm font-semibold mb-2">
              {t("applications.matchScore.missingKeywords")}
            </h4>
            <div className="flex flex-wrap gap-2">
              {data.missing_keywords.map((keyword) => (
                <span
                  key={keyword}
                  className="inline-flex items-center rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-medium text-amber-800 dark:bg-amber-900 dark:text-amber-200"
                >
                  {keyword}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Summary */}
        {data.summary && (
          <div>
            <h4 className="text-sm font-semibold mb-2">
              {t("applications.matchScore.summary")}
            </h4>
            <p className="text-sm text-muted-foreground">{data.summary}</p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
