import os
import google.generativeai as genai


class GeminiClient:
    def __init__(self):
        api_key = os.getenv("GEMINI_API_KEY", "")
        genai.configure(api_key=api_key)
        self.model = genai.GenerativeModel("gemini-1.5-flash")

    async def analyze(self, trade_summary: dict) -> str:
        prompt = f"""Ты финансовый аналитик торговой платформы. Проанализируй результат торговой операции и дай краткий комментарий (2-3 предложения).

Данные операции:
- Символы: {trade_summary.get('symbols', [])}
- Тренд рынка: {trade_summary.get('trend', 'unknown')}
- Рекомендация: {trade_summary.get('recommendation', 'HOLD')}
- Уровень риска: {trade_summary.get('risk_level', 'MEDIUM')}
- Статус сделки: {trade_summary.get('trade_status', 'unknown')}
- Цена исполнения: {trade_summary.get('execution_price', 0)}

Дай краткий профессиональный комментарий на русском языке."""

        try:
            response = await self.model.generate_content_async(prompt)
            return response.text
        except Exception as e:
            return f"Анализ недоступен: {str(e)}"
