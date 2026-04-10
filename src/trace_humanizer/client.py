from __future__ import annotations

from typing import Any, Sequence
import httpx
from phoenix.client import Client as PhoenixClient_


class PhoenixClient:
    def __init__(self, base_url: str, api_key: str | None = None):
        self._base_url = base_url.rstrip("/")
        self._api_key = api_key
        self._client = PhoenixClient_(base_url=base_url, api_key=api_key)
        self._http_client = httpx.Client(base_url=base_url, timeout=30.0)

    def _headers(self) -> dict[str, str]:
        headers = {"Content-Type": "application/json", "accept": "application/json"}
        if self._api_key:
            headers["authorization"] = f"Bearer {self._api_key}"
        return headers

    def list_projects(self) -> list[dict]:
        projects = self._client.projects.list_projects()
        return [p.model_dump() if hasattr(p, "model_dump") else dict(p) for p in projects]

    def get_project(self, project_id: str) -> dict | None:
        try:
            project = self._client.projects.get_project(project_id)
            return project.model_dump() if hasattr(project, "model_dump") else dict(project)
        except Exception:
            return None

    def get_spans(self, project_identifier: str, trace_ids: Sequence[str] | None = None, limit: int = 10000) -> list[dict]:
        spans = self._client.spans.get_spans(
            project_identifier=project_identifier,
            trace_ids=trace_ids,
            limit=limit,
        )
        return [self._span_to_dict(s) for s in spans]

    def get_span(self, project_id: str, span_id: str) -> dict | None:
        try:
            resp = self._http_client.get(
                f"/v1/projects/{project_id}/spans",
                headers=self._headers(),
                params={"span_id": span_id},
            )
            if resp.status_code == 404:
                return None
            resp.raise_for_status()
            data = resp.json()
            if isinstance(data, dict) and "data" in data:
                items = data["data"]
                return items[0] if items else None
            return data if isinstance(data, list) else data
        except Exception:
            return None

    def get_trace_spans(self, project_identifier: str, trace_id: str) -> list[dict]:
        return self.get_spans(project_identifier, trace_ids=[trace_id])

    def _span_to_dict(self, span: Any) -> dict:
        if hasattr(span, "model_dump"):
            d = span.model_dump()
        elif hasattr(span, "dict"):
            d = span.dict()
        else:
            d = dict(span)
        return d

    def close(self):
        self._http_client.close()
