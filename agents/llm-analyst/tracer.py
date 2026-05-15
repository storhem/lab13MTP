import os
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource


def init_tracer(service_name: str) -> trace.Tracer:
    """Initialise OpenTelemetry with an OTLP HTTP exporter and return a named tracer."""
    endpoint = os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://jaeger:4318")
    # OTLPSpanExporter expects the /v1/traces path
    otlp_endpoint = endpoint.rstrip("/") + "/v1/traces"

    exporter = OTLPSpanExporter(endpoint=otlp_endpoint)
    processor = BatchSpanProcessor(exporter)

    resource = Resource.create({"service.name": service_name})
    provider = TracerProvider(resource=resource)
    provider.add_span_processor(processor)

    trace.set_tracer_provider(provider)
    return trace.get_tracer(service_name)
