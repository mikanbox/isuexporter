package isuexporter

import (
	"context"
	"encoding/json"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"os"
)

type FileSpanExporter struct {
	file *os.File
}

func NewFileSpanExporter(filename string) (*FileSpanExporter, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	return &FileSpanExporter{file: file}, nil
}

func (f *FileSpanExporter) Shutdown(ctx context.Context) error {
	return f.file.Close()
}

func TraceFileProvider(filePath string, serviceName string, serviceVersion string) (func(), error) {
	// otelのライブラリではexporterという枠組みで計測した情報をどこに送信するかを設定できる
	// 今回は標準出力(stderr)に出力するためのexporterを作成する

	exporter, err := NewFileSpanExporter(filePath)

	if err != nil {
		// exporterの作成に失敗した場合のエラー処理
	}

	// リソースは、OpenTelemetryのデータに付加するメタデータを定義する
	// ここでは、スキーマURL、サービス名、サービスバージョンをメタデータとして設定している
	otelResource := resource.NewWithAttributes(
		semconv.SchemaURL,                                // スキーマURLを設定
		semconv.ServiceNameKey.String(serviceName),       // サービス名を設定
		semconv.ServiceVersionKey.String(serviceVersion), // サービスバージョンを設定
	)

	// TracerProviderはOpenTelemetryのトレースデータを処理するコンポーネント
	// ここでは、作成したexporterとリソースを設定している
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),                // 作成したexporterを設定
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // すべてのスパンをサンプリングする
		sdktrace.WithResource(otelResource),           // 設定したリソースを適用
	)
	// TracerProviderをOpenTelemetryのグローバル設定に登録する
	otel.SetTracerProvider(tracerProvider)

	// TracerProviderの終了処理を行う関数を作成する
	cleanup := func() {
		ctx, cancel := context.WithCancel(context.Background()) // コンテキストを作成
		defer cancel()                                          // コンテキストをキャンセルする
		err := tracerProvider.Shutdown(ctx)
		if err != nil {
			return
		} // TracerProviderをシャットダウンする
	}
	// cleanupは、アプリケーションの終了時に呼び出される必要がある
	// これにより、TracerProviderがクリーンアップされ、リソースが適切に解放される
	return cleanup, nil
}

// ExportSpans メソッドを実装（SpanExporter インターフェースを満たす）
func (f *FileSpanExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		// スパンデータを JSON 形式で出力
		data, err := json.MarshalIndent(spanToMap(span), "", "  ")
		if err != nil {
			return err
		}
		_, err = f.file.Write(append(data, '\n'))
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadOnlySpan から簡易なマップに変換する（必要に応じてカスタマイズ可能）
func spanToMap(span sdktrace.ReadOnlySpan) map[string]interface{} {
	return map[string]interface{}{
		"name":       span.Name(),
		"startTime":  span.StartTime(),
		"endTime":    span.EndTime(),
		"attributes": span.Attributes(),
	}
}
