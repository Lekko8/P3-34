#!/usr/bin/env python3
# -*- coding: utf-8 -*-

from __future__ import annotations

import datetime as dt
import pathlib
import re
import time
import subprocess
import winsound

import serial


# =========================
# НАСТРОЙКИ
# =========================

PORT = "COM4"
SAVE_DIR = r"C:/Users/admin/Desktop/p3-34/P3-34 measurments"

BAUDRATE = 9600
TIMEOUT = 2.0

INTERVAL_MIN = 2
CONNECT_DELAY_SEC = 2.0
RECONNECT_DELAY_SEC = 5.0
COMMAND_RESPONSE_TIMEOUT_SEC = 8.0

# Звуковой сигнал при остановке по порогу
ENABLE_SOUND_ON_STOP = True
SOUND_VOLUME = 100
SOUND_FREQ_HZ = 2000
SOUND_DURATION_MS = 500
SOUND_REPEAT = 5
SOUND_PAUSE_SEC = 0.2

CSV_SEPARATOR = ";"

# None -> широкополосный режим прибора
# 2.2 -> измерение только на 2,2 ГГц
FREQ_GHZ = None

# Остановить скрипт, если СРЕДНЕЕ значение ниже этого порога
STOP_THRESHOLD = 0.5


EOT_BYTE = b"\x04"


class P334Error(Exception):
    pass


class P334:
    def __init__(self, port: str, baudrate: int = 9600, timeout: float = 2.0):
        self.port = port
        self.baudrate = baudrate
        self.timeout = timeout
        self.ser: serial.Serial | None = None

    def connect(self) -> None:
        self.close()

        self.ser = serial.Serial(
            port=self.port,
            baudrate=self.baudrate,
            timeout=self.timeout,
            write_timeout=None,
        )

        self.ser.dtr = False
        self.ser.rts = False

        time.sleep(CONNECT_DELAY_SEC)
        self.reset_buffers()

    def reconnect(self) -> None:
        self.close()
        self.connect()

    def close(self) -> None:
        if self.ser is not None:
            try:
                if self.ser.is_open:
                    self.ser.close()
            except Exception:
                pass
        self.ser = None

    def reset_buffers(self) -> None:
        ser = self._ensure_open()

        try:
            ser.reset_input_buffer()
        except Exception:
            pass

        try:
            ser.reset_output_buffer()
        except Exception:
            pass

        deadline = time.time() + 0.3
        while time.time() < deadline:
            try:
                waiting = ser.in_waiting
            except Exception:
                break

            if not waiting:
                break

            try:
                ser.read(waiting)
            except Exception:
                break

            time.sleep(0.02)

    def _ensure_open(self) -> serial.Serial:
        if self.ser is None or not self.ser.is_open:
            raise P334Error("Соединение с прибором не открыто.")
        return self.ser

    def _clear_input(self) -> None:
        ser = self._ensure_open()
        try:
            ser.reset_input_buffer()
        except Exception:
            pass

    def send_command(self, command: str) -> str:
        ser = self._ensure_open()
        self._clear_input()

        if not command.endswith("\r\n"):
            command += "\r\n"

        data = command.encode("ascii")

        try:
            n = ser.write(data)
            ser.flush()
        except (serial.SerialException, OSError) as e:
            raise P334Error(f"Ошибка записи в COM-порт {self.port}: {e}") from e

        if n != len(data):
            raise P334Error(f"Команда записана не полностью: {n} из {len(data)} байт.")

        return self._read_until_response()

    def _read_until_response(self) -> str:
        ser = self._ensure_open()
        started = time.time()
        buffer = bytearray()

        while True:
            if time.time() - started > COMMAND_RESPONSE_TIMEOUT_SEC:
                raise P334Error(
                    f"Нет ответа от прибора более {COMMAND_RESPONSE_TIMEOUT_SEC} сек."
                )

            try:
                chunk = ser.read(1)
            except (serial.SerialException, OSError) as e:
                raise P334Error(f"Ошибка чтения из COM-порта {self.port}: {e}") from e

            if chunk:
                buffer.extend(chunk)

                if EOT_BYTE in buffer:
                    buffer = buffer.split(EOT_BYTE)[0]
                    break

                if b"ready >>>" in buffer:
                    break
            else:
                time.sleep(0.05)

        text = buffer.decode("utf-8", errors="replace")
        text = cleanup_response(text)

        if not text:
            raise P334Error("Получен пустой ответ от прибора.")

        return text

    def get_id(self) -> str:
        return self.send_command("?")

    def get_battery(self) -> str:
        return self.send_command("UBATT")

    def test(self) -> str:
        return self.send_command("TEST")

    def new_measurements(self) -> str:
        return self.send_command("NEWMEAS")

    def measure_average(self) -> str:
        return self.send_command("MEAS:AVR")

    def measure_max(self) -> str:
        return self.send_command("MEAS:MAX")


def cleanup_response(text: str) -> str:
    text = text.replace("\x04", "")
    text = text.replace("EOT", "")
    text = text.replace("ready >>>", "")
    text = text.replace("\r", "\n")

    lines = []
    for line in text.split("\n"):
        cleaned = line.strip()
        if cleaned:
            lines.append(cleaned)

    return " ".join(lines).strip()


def is_battery_response(text: str) -> bool:
    t = cleanup_response(text).upper()
    return "UBATT" in t or (" V " in f" {t} " and "SMAX" not in t and "<S>" not in t)


def is_average_response(text: str) -> bool:
    return "<S>" in cleanup_response(text)


def is_max_response(text: str) -> bool:
    return "SMAX" in cleanup_response(text).upper()


def parse_measurement(text: str) -> tuple[float | None, float | None, str]:
    cleaned = cleanup_response(text)

    numbers = re.findall(r"[0-9]+(?:[.,][0-9]+)?", cleaned)
    if len(numbers) < 2:
        return None, None, ""

    value = float(numbers[0].replace(",", "."))
    error = float(numbers[1].replace(",", "."))

    unit_match = re.search(r"(uW/cm2|µW/cm2|mW/cm2|W/m2)", cleaned, re.IGNORECASE)
    unit = unit_match.group(1) if unit_match else ""

    return value, error, unit


def should_stop_by_average(average_value: float | None) -> bool:
    return average_value is not None and average_value < STOP_THRESHOLD


def ensure_output_dir(path_str: str) -> pathlib.Path:
    path = pathlib.Path(path_str)
    path.mkdir(parents=True, exist_ok=True)
    return path


def make_output_file_path(save_dir: pathlib.Path) -> pathlib.Path:
    now = dt.datetime.now()
    return save_dir / now.strftime("p3_34_%Y%m%d_%H%M%S.csv")


def csv_escape(value: object) -> str:
    text = "" if value is None else str(value)
    text = text.replace('"', '""')
    return f'"{text}"'


def write_csv_line(file_obj, values: list[object]) -> None:
    file_obj.write(CSV_SEPARATOR.join(csv_escape(v) for v in values) + "\n")


def format_number_for_csv(value: float | None) -> str:
    if value is None:
        return ""
    text = f"{value:.6f}".rstrip("0").rstrip(".")
    return text.replace(".", ",")


def format_freq_for_header(freq_ghz: float | None) -> str:
    if freq_ghz is None:
        return ""
    return f"{str(freq_ghz).replace('.', ',')} ГГц"


def set_volume(percent: int) -> None:
    percent = max(0, min(100, percent))
    scalar = percent / 100

    command = f"""
    Add-Type -TypeDefinition @"
    using System;
    using System.Runtime.InteropServices;

    [Guid("5CDF2C82-841E-4546-9722-0CF74078229A")]
    [InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    interface IAudioEndpointVolume {{
        int RegisterControlChangeNotify(IntPtr pNotify);
        int UnregisterControlChangeNotify(IntPtr pNotify);
        int GetChannelCount(out uint pnChannelCount);
        int SetMasterVolumeLevel(float fLevelDB, Guid pguidEventContext);
        int SetMasterVolumeLevelScalar(float fLevel, Guid pguidEventContext);
        int GetMasterVolumeLevel(out float pfLevelDB);
        int GetMasterVolumeLevelScalar(out float pfLevel);
        int SetChannelVolumeLevel(uint nChannel, float fLevelDB, Guid pguidEventContext);
        int SetChannelVolumeLevelScalar(uint nChannel, float fLevel, Guid pguidEventContext);
        int GetChannelVolumeLevel(uint nChannel, out float pfLevelDB);
        int GetChannelVolumeLevelScalar(uint nChannel, out float pfLevel);
        int SetMute(bool bMute, Guid pguidEventContext);
        int GetMute(out bool pbMute);
        int GetVolumeStepInfo(out uint pnStep, out uint pnStepCount);
        int VolumeStepUp(Guid pguidEventContext);
        int VolumeStepDown(Guid pguidEventContext);
        int QueryHardwareSupport(out uint pdwHardwareSupportMask);
        int GetVolumeRange(out float pflVolumeMindB, out float pflVolumeMaxdB, out float pflVolumeIncrementdB);
    }}

    [Guid("BCDE0395-E52F-467C-8E3D-C4579291692E")]
    [ComImport]
    class MMDeviceEnumerator {{}}

    [Guid("A95664D2-9614-4F35-A746-DE8DB63617E6")]
    [InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    interface IMMDeviceEnumerator {{
        int NotImpl1();
        int GetDefaultAudioEndpoint(int dataFlow, int role, out IMMDevice ppDevice);
    }}

    [Guid("D666063F-1587-4E43-81F1-B948E807363F")]
    [InterfaceType(ComInterfaceType.InterfaceIsIUnknown)]
    interface IMMDevice {{
        int Activate(ref Guid iid, int dwClsCtx, IntPtr pActivationParams, out IAudioEndpointVolume ppInterface);
    }}

    public class Audio {{
        public static void SetVolume(float level) {{
            IMMDeviceEnumerator enumerator = (IMMDeviceEnumerator)(new MMDeviceEnumerator());
            IMMDevice device;
            enumerator.GetDefaultAudioEndpoint(0, 1, out device);

            IAudioEndpointVolume volume;
            Guid guid = typeof(IAudioEndpointVolume).GUID;
            device.Activate(ref guid, 23, IntPtr.Zero, out volume);

            volume.SetMute(false, Guid.Empty);
            volume.SetMasterVolumeLevelScalar(level, Guid.Empty);
        }}
    }}
"@

    [Audio]::SetVolume({scalar})
    """

    subprocess.run(
        ["powershell", "-Command", command],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )


def play_stop_signal() -> None:
    if not ENABLE_SOUND_ON_STOP:
        return

    melody = [
        (784, 600),
        (784, 600),
        (784, 600),
        (622, 400),
        (932, 300),
        (784, 600),
        (622, 400),
        (932, 300),
        (784, 800),
    ]

    try:
        set_volume(SOUND_VOLUME)

        for freq, duration in melody:
            winsound.Beep(freq, duration)
            time.sleep(0.1)

    except Exception as e:
        print(f"Не удалось воспроизвести звуковой сигнал: {e}")


def set_frequency_if_needed(device: P334) -> None:
    if FREQ_GHZ is None:
        print("\n[MODE] Широкополосный режим прибора")
        return

    print(f"\n[SET FREQ] {str(FREQ_GHZ).replace('.', ',')} ГГц")
    reply = device.send_command(f"FREQ:{FREQ_GHZ}")
    print(reply)


def write_file_header(
    file_path: pathlib.Path,
    started_at: str,
    port: str,
    interval_min: float,
    freq_ghz: float | None,
    device_id: str,
    battery: str,
    test_result: str,
) -> None:
    with file_path.open("w", encoding="utf-8-sig", newline="") as f:
        write_csv_line(f, ["P3-34 measurement log"])
        write_csv_line(f, ["Started at", started_at])
        write_csv_line(f, ["Port", port])
        write_csv_line(f, ["Interval min", str(interval_min).replace(".", ",")])
        write_csv_line(f, ["Frequency", format_freq_for_header(freq_ghz)])
        write_csv_line(f, ["Stop if average below", format_number_for_csv(STOP_THRESHOLD)])
        write_csv_line(f, [])
        write_csv_line(f, ["Device ID", device_id])
        write_csv_line(f, ["Battery", battery])
        write_csv_line(f, ["Test", test_result])
        write_csv_line(f, [])
        write_csv_line(
            f,
            [
                "interval_start",
                "interval_end",
                "battery",
                "average_value",
                "average_error",
                "max_value",
                "max_error",
                "unit",
                "status",
                "error_message",
            ],
        )


def append_csv_row(
    file_path: pathlib.Path,
    interval_started: str,
    interval_finished: str,
    battery="",
    average_value: float | None = None,
    average_error: float | None = None,
    max_value: float | None = None,
    max_error: float | None = None,
    unit="",
    status="OK",
    error_message="",
) -> None:
    with file_path.open("a", encoding="utf-8-sig", newline="") as f:
        write_csv_line(
            f,
            [
                interval_started,
                interval_finished,
                battery,
                format_number_for_csv(average_value),
                format_number_for_csv(average_error),
                format_number_for_csv(max_value),
                format_number_for_csv(max_error),
                unit,
                status,
                error_message,
            ],
        )


def wait_interval(interval_seconds: int) -> None:
    end_time = time.time() + interval_seconds
    while time.time() < end_time:
        time.sleep(min(1.0, max(0.0, end_time - time.time())))


def safe_get_battery(device: P334) -> str:
    resp = device.get_battery()
    if not is_battery_response(resp):
        raise P334Error(f"Получен неожиданный ответ вместо UBATT: {resp}")
    return resp


def collect_measurements(device: P334, rounds: int = 3):
    average_value = None
    average_error = None
    max_value = None
    max_error = None
    unit = ""

    for _ in range(rounds):
        if average_value is None:
            resp = device.measure_average()
            if is_average_response(resp):
                average_value, average_error, avg_unit = parse_measurement(resp)
                unit = unit or avg_unit
            elif is_max_response(resp):
                max_value, max_error, max_unit = parse_measurement(resp)
                unit = unit or max_unit
            else:
                raise P334Error(f"Неожиданный ответ на MEAS:AVR: {resp}")

        if max_value is None:
            resp = device.measure_max()
            if is_max_response(resp):
                max_value, max_error, max_unit = parse_measurement(resp)
                unit = unit or max_unit
            elif is_average_response(resp):
                average_value, average_error, avg_unit = parse_measurement(resp)
                unit = unit or avg_unit
            else:
                raise P334Error(f"Неожиданный ответ на MEAS:MAX: {resp}")

        if average_value is not None and max_value is not None:
            break

    return average_value, average_error, max_value, max_error, unit


def connect_and_initialize(device: P334):
    device.connect()
    device.reset_buffers()
    device_id = device.get_id()
    battery = safe_get_battery(device)
    test_result = device.test()
    return device_id, battery, test_result


def reconnect_loop(device: P334):
    while True:
        try:
            print(f"\nПытаюсь переподключиться к {device.port} ...")
            device.reconnect()
            device.reset_buffers()

            device_id = device.get_id()
            battery = safe_get_battery(device)
            test_result = device.test()

            set_frequency_if_needed(device)
            device.reset_buffers()

            print("Переподключение успешно.")
            return device_id, battery, test_result

        except Exception as e:
            print(f"Переподключение не удалось: {e}")
            time.sleep(RECONNECT_DELAY_SEC)


def try_get_measurements_after_reconnect(device: P334):
    try:
        device.reset_buffers()
        return collect_measurements(device, rounds=3)
    except Exception:
        return None, None, None, None, ""


def main() -> None:
    interval_seconds = int(INTERVAL_MIN * 60)

    save_dir = ensure_output_dir(SAVE_DIR)
    output_path = make_output_file_path(save_dir)
    started_at = dt.datetime.now().strftime("%Y-%m-%d %H:%M:%S")

    print(f"Порт: {PORT}")
    print(f"Папка сохранения: {save_dir}")
    print(f"Файл результатов: {output_path}")
    print(f"Интервал измерения: {str(INTERVAL_MIN).replace('.', ',')} мин")
    print(f"Порог остановки по среднему: {str(STOP_THRESHOLD).replace('.', ',')}")

    device = P334(PORT, baudrate=BAUDRATE, timeout=TIMEOUT)

    try:
        device_id, battery0, test_result = connect_and_initialize(device)
        set_frequency_if_needed(device)
        device.reset_buffers()

        write_file_header(
            file_path=output_path,
            started_at=started_at,
            port=PORT,
            interval_min=INTERVAL_MIN,
            freq_ghz=FREQ_GHZ,
            device_id=device_id,
            battery=battery0,
            test_result=test_result,
        )

        print("\nСоздан CSV-файл результатов.")

        cycle_number = 1

        while True:
            interval_started = dt.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
            battery_now = ""

            try:
                print(f"\n=== ЦИКЛ #{cycle_number} ===")
                print(f"Старт интервала: {interval_started}")

                battery_now = safe_get_battery(device)
                print("[UBATT]")
                print(battery_now)

                start_reply = device.new_measurements()
                print("[NEWMEAS]")
                print(start_reply)

                wait_interval(interval_seconds)

                average_value, average_error, max_value, max_error, unit = collect_measurements(
                    device, rounds=3
                )

                interval_finished = dt.datetime.now().strftime("%Y-%m-%d %H:%M:%S")

                append_csv_row(
                    file_path=output_path,
                    interval_started=interval_started,
                    interval_finished=interval_finished,
                    battery=battery_now,
                    average_value=average_value,
                    average_error=average_error,
                    max_value=max_value,
                    max_error=max_error,
                    unit=unit,
                    status="OK",
                    error_message="",
                )

                print(
                    f"Сохранено: AVG={average_value}±{average_error}; "
                    f"MAX={max_value}±{max_error} {unit}"
                )

                if should_stop_by_average(average_value):
                    stop_time = dt.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                    message = f"Среднее значение {average_value:.3f} ниже порога {STOP_THRESHOLD}"

                    append_csv_row(
                        file_path=output_path,
                        interval_started=stop_time,
                        interval_finished=stop_time,
                        battery=battery_now,
                        status="STOP",
                        error_message=message,
                    )

                    print(f"\n{message}. Скрипт остановлен.")
                    play_stop_signal()
                    break

                cycle_number += 1

            except KeyboardInterrupt:
                raise

            except Exception as e:
                interval_finished = dt.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                error_text = f"Ошибка связи: {e}"

                print(f"\n{error_text}")
                print("Пытаюсь переподключиться и дочитать AVR/MAX...")

                device_id, battery_reconnected, test_result = reconnect_loop(device)

                average_value, average_error, max_value, max_error, unit = (
                    try_get_measurements_after_reconnect(device)
                )

                battery_to_write = (
                    battery_now if is_battery_response(battery_now) else battery_reconnected
                )

                append_csv_row(
                    file_path=output_path,
                    interval_started=interval_started,
                    interval_finished=interval_finished,
                    battery=battery_to_write,
                    average_value=average_value,
                    average_error=average_error,
                    max_value=max_value,
                    max_error=max_error,
                    unit=unit,
                    status="ERROR",
                    error_message=error_text,
                )

                if should_stop_by_average(average_value):
                    stop_time = dt.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
                    message = f"Среднее значение {average_value:.3f} ниже порога {STOP_THRESHOLD} после переподключения"

                    append_csv_row(
                        file_path=output_path,
                        interval_started=stop_time,
                        interval_finished=stop_time,
                        battery=battery_to_write,
                        status="STOP",
                        error_message=message,
                    )

                    print(f"\n{message}. Скрипт остановлен.")
                    play_stop_signal()
                    break

                print("Строка с ERROR записана в файл.")
                cycle_number += 1

    except KeyboardInterrupt:
        print("\nСкрипт остановлен пользователем.")
        print(f"Последние данные сохранены в: {output_path}")

    finally:
        device.close()


if __name__ == "__main__":
    main()
