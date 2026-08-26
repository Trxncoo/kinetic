#!/usr/bin/env python3
"""Parse `go test -bench` output (from any Go package) and write one
bench/<Subject>/ folder per benchmark subject, each with a chart.svg line
chart (ns/op vs GOMAXPROCS) and a summary.txt. Benchmarks are grouped by
subject — the token right after "Benchmark", e.g. both BenchmarkBus_Publish
and BenchmarkBus_Publish_Parallel belong to "Bus" — so a newly added
benchmark with a new subject gets its own folder for free, no script
changes needed. These folders are meant to be committed, to track
performance over time via git history."""

import os
import re
import statistics
import sys

LINE_RE = re.compile(
    r"^(Benchmark\S+?)(?:-(\d+))?\s+\d+\s+([\d.]+)\s+ns/op"
)

COLORS = ["#3b82f6", "#ef4444", "#10b981", "#f59e0b", "#8b5cf6", "#ec4899"]


def subject_of(name):
    # "BenchmarkBus_Publish" -> "Bus"; "BenchmarkBus_Publish_Parallel" -> "Bus"
    rest = name[len("Benchmark"):]
    return rest.split("_", 1)[0]


def parse(lines):
    # (name, cpu) -> [ns/op, ...]
    samples = {}
    for line in lines:
        m = LINE_RE.match(line)
        if not m:
            continue
        name, cpu, ns = m.group(1), m.group(2), float(m.group(3))
        cpu = int(cpu) if cpu else 1
        samples.setdefault((name, cpu), []).append(ns)
    return samples


def to_series(samples):
    # name -> {cpu: (mean, min, max)}
    series = {}
    for (name, cpu), values in samples.items():
        series.setdefault(name, {})[cpu] = (
            statistics.mean(values),
            min(values),
            max(values),
        )
    return series


def group_by_subject(series):
    groups = {}
    for name, cpus in series.items():
        groups.setdefault(subject_of(name), {})[name] = cpus
    return groups


def render_svg(series, plot_width=760, height=460, pad=60, legend_w=200):
    width = plot_width + legend_w
    all_cpus = sorted({cpu for cpus in series.values() for cpu in cpus})
    all_means = [v[0] for cpus in series.values() for v in cpus.values()]
    y_max = max(all_means) * 1.15
    y_min = 0.0

    def x_of(cpu):
        i = all_cpus.index(cpu)
        if len(all_cpus) == 1:
            return pad
        return pad + i * (plot_width - 2 * pad) / (len(all_cpus) - 1)

    def y_of(ns):
        return height - pad - (ns - y_min) / (y_max - y_min) * (height - 2 * pad)

    svg = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" '
        f'font-family="sans-serif" font-size="12">',
        f'<rect width="{width}" height="{height}" fill="white"/>',
    ]

    # gridlines + y labels
    for i in range(6):
        ns = y_max * i / 5
        y = y_of(ns)
        svg.append(
            f'<line x1="{pad}" y1="{y:.1f}" x2="{plot_width - pad}" y2="{y:.1f}" '
            f'stroke="#e5e7eb"/>'
        )
        svg.append(f'<text x="{pad - 8}" y="{y + 4:.1f}" text-anchor="end">{ns:.1f}</text>')

    # x axis labels
    for cpu in all_cpus:
        x = x_of(cpu)
        svg.append(f'<text x="{x:.1f}" y="{height - pad + 20}" text-anchor="middle">cpu={cpu}</text>')

    svg.append(f'<text x="{pad - 40}" y="{pad - 20}">ns/op</text>')

    # series
    for i, (name, cpus) in enumerate(sorted(series.items())):
        color = COLORS[i % len(COLORS)]
        pts = sorted(cpus.items())
        points = " ".join(f"{x_of(c):.1f},{y_of(v[0]):.1f}" for c, v in pts)
        svg.append(f'<polyline points="{points}" fill="none" stroke="{color}" stroke-width="2"/>')
        for c, (mean, lo, hi) in pts:
            x = x_of(c)
            svg.append(f'<line x1="{x:.1f}" y1="{y_of(lo):.1f}" x2="{x:.1f}" y2="{y_of(hi):.1f}" stroke="{color}" stroke-width="1"/>')
            svg.append(f'<circle cx="{x:.1f}" cy="{y_of(mean):.1f}" r="3" fill="{color}"/>')

        # legend
        ly = pad + i * 18
        svg.append(f'<line x1="{plot_width + 10}" y1="{ly}" x2="{plot_width + 30}" y2="{ly}" stroke="{color}" stroke-width="2"/>')
        svg.append(f'<text x="{plot_width + 34}" y="{ly + 4}">{name}</text>')

    svg.append("</svg>")
    return "\n".join(svg)


def render_txt(series):
    lines = []
    for name, cpus in sorted(series.items()):
        for cpu, (mean, lo, hi) in sorted(cpus.items()):
            lines.append(f"{name}-{cpu}: mean={mean:.2f} ns/op min={lo:.2f} max={hi:.2f}")
    return "\n".join(lines) + "\n"


def main():
    in_path = sys.argv[1] if len(sys.argv) > 1 and sys.argv[1] != "-" else None

    if in_path is None and sys.stdin.isatty():
        print(f"usage: go test ... | {sys.argv[0]} [bench-output.txt] [out-dir]", file=sys.stderr)
        sys.exit(1)

    out_dir = sys.argv[2] if len(sys.argv) > 2 else "bench"

    if in_path:
        with open(in_path) as f:
            samples = parse(f.readlines())
    else:
        samples = parse(sys.stdin)

    if not samples:
        print("no benchmark lines found", file=sys.stderr)
        sys.exit(1)

    series = to_series(samples)
    groups = group_by_subject(series)

    for subject, group in sorted(groups.items()):
        subject_dir = os.path.join(out_dir, subject)
        os.makedirs(subject_dir, exist_ok=True)

        legend_w = 40 + 8 * max(len(name) for name in group)
        svg_path = os.path.join(subject_dir, "chart.svg")
        txt_path = os.path.join(subject_dir, "summary.txt")

        with open(svg_path, "w") as f:
            f.write(render_svg(group, legend_w=legend_w))
        with open(txt_path, "w") as f:
            f.write(render_txt(group))

        print(f"wrote {svg_path}, {txt_path}")


if __name__ == "__main__":
    main()
