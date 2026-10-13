#!/usr/bin/env python3
"""从 OpenRouter 拉取最新模型元数据，合并进前端内置 model-capabilities.json。

用法（仓库根目录执行，需可直连 openrouter.ai）:
    python scripts/update_model_caps.py

合并策略：内置条目保留（视为人工校验过），仅追加内置表缺失的模型；
输出按 key 排序、UTF-8、两空格缩进。
"""

import json
import sys
import time
import urllib.request
from pathlib import Path

OPENROUTER_URL = "https://openrouter.ai/api/v1/models"
TARGET = Path(__file__).resolve().parent.parent / "frontend" / "src" / "config" / "model-capabilities.json"


def fetch_models(timeout: int = 30) -> dict:
    req = urllib.request.Request(OPENROUTER_URL, headers={"User-Agent": "Cue-ModelCaps/1.0"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        if resp.status != 200:
            sys.exit(f"OpenRouter 返回 {resp.status}")
        data = json.load(resp)["data"]

    models = {}
    for m in data:
        model_id = m.get("id", "")
        parts = model_id.split("/", 1)
        if len(parts) != 2 or not all(parts):
            continue
        org, slug = parts
        arch = m.get("architecture", {})
        inputs = arch.get("input_modalities") or []
        models[slug.lower()] = {
            "provider": org,
            "name": m.get("name", slug),
            "description": m.get("description", ""),
            "supports_image": "image" in inputs or None,
            "context_length": m.get("context_length", 0),
            "inputs": inputs or None,
            "outputs": arch.get("output_modalities") or None,
        }
    # 清掉值为 None 的可选键，与内置格式一致
    cleaned = {}
    for slug, e in models.items():
        cleaned[slug] = {k: v for k, v in e.items() if v is not None}
    return cleaned


def main() -> None:
    fetched = fetch_models()
    print(f"OpenRouter 返回 {len(fetched)} 个模型")

    existing = json.loads(TARGET.read_text(encoding="utf-8"))
    added = sorted(set(fetched) - set(existing))
    for slug in added:
        existing[slug] = fetched[slug]

    TARGET.write_text(
        json.dumps(existing, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )
    print(f"内置表原有 {len(existing) - len(added)} 条，新增 {len(added)} 条 -> {TARGET}")
    if added:
        print("新增模型:", ", ".join(added[:20]) + ("..." if len(added) > 20 else ""))


if __name__ == "__main__":
    start = time.time()
    main()
    print(f"完成，耗时 {time.time() - start:.1f}s")
