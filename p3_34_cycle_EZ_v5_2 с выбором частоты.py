#!/usr/bin/env python3
# -*- coding: utf-8 -*-

from __future__ import annotations

import datetime as dt
import pathlib
import re
import time

import serial


# =========================
# НАСТРОЙКИ
# =========================

PORT = "COM4"
SAVE_DIR = r"C:/Users/GUEST/Documents/P3-34/P3-34 measurments"

BAUDRATE = 9600
TIMEOUT = 2.0

# Длительность одного интервала измерения в минутах
INTERVAL_MIN = 2

# Пауза после открытия порта
CONNECT_DELAY_SEC = 2.0

# Через сколько секунд пытаться переподключиться при обрыве
RECONNECT_DELAY_SEC = 5.0

# Максимальное время ожидания ответа на одну команду
COMMAND_RESPONSE_TIMEOUT_SEC = 8.0

# Разделитель CSV
CSV_SEPARATOR = ";"

# Частота в ГГц.
# None -> широкополосный режим прибора
# Например 2.2 -> измерение только на 2.2 ГГц
FREQ_GHZ = 2.2


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

        # Для Bluetooth COM на Windows это часто помогает
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

        # дочищаем возможный хвост в буфере
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
    t = cleanup_response(text)
    return "<S>" in t


def is_max_response(text: str) -> bool:
    t = cleanup_response(text).upper()
    return "SMAX" in t


def parse_measurement(text: str) -> tuple[float | None, float | None, str]:
    """
    Разбирает строки типа:
    <S> = (0.324 ± 0.178) uW/cm2 (P=0.95)
    Smax = (4.389 ± 0.214) uW/cm2 (P=0.95)
    """
    cleaned = cleanup_response(text)

    numbers = re.findall(r"[0-9]+(?:[.,][0-9]+)?", cleaned)
    if len(numbers) < 2:
        return None, None, ""

    value = float(numbers[0].replace(",", "."))
    error = float(numbers[1].replace(",", "."))

    unit_match = re.search(r"(uW/cm2|µW/cm2|mW/cm2|W/m2)", cleaned, re.IGNORECASE)
    unit = unit_match.group(1) if unit_match else ""

    return value, error, unit


def ensure_output_dir(path_str: str) -> pathlib.Path:
    path = pathlib.Path(path_str)
    path.mkdir(parents=True, exist_ok=True)
    return path


def make_output_file_path(save_dir: pathlib.Path) -> pathlib.Path:
    now = dt.datetime.now()
    filename = now.strftime("p3_34_%Y%m%d_%H%M%S.csv")
    return save_dir / filename


def csv_escape(value: object) -> str:
    text = "" if value is None else str(value)
    text = text.replace('"', '""')
    return f'"{text}"'


def write_csv_line(file_obj, values: list[object]) -> None:
    line = CSV_SEPARATOR.join(csv_escape(v) for v in values)
    file_obj.write(line + "\n")


def format_number_for_csv(value: float | None) -> str:
    """
    Пишем числа с запятой для Excel на русском Windows.
    """
    if value is None:
        return ""
    text = f"{value:.6f}".rstrip("0").rstrip(".")
    return text.replace(".", ",")


def format_freq_for_header(freq_ghz: float | None) -> str:
    if freq_ghz is None:
        return ""
    return f"{str(freq_ghz).replace('.', ',')} ГГц"


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
        remaining = end_time - time.time()
        time.sleep(min(1.0, max(0.0, remaining)))


def safe_get_battery(device: P334) -> str:
    resp = device.get_battery()
    if not is_battery_response(resp):
        raise P334Error(f"Получен неожиданный ответ вместо UBATT: {resp}")
    return resp


def collect_measurements(device: P334, rounds: int = 3) -> tuple[
    float | None, float | None, float | None, float | None, str
]:
    """
    Надёжно собирает AVR и MAX даже если ответы "съехали":
    например, на запрос AVR пришёл Smax.
    """
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


def connect_and_initialize(device: P334) -> tuple[str, str, str]:
    device.connect()
    device.reset_buffers()
    device_id = device.get_id()
    battery = safe_get_battery(device)
    test_result = device.test()
    return device_id, battery, test_result


def reconnect_loop(device: P334) -> tuple[str, str, str]:
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
            print("[ID]")
            print(device_id)
            print("[UBATT]")
            print(battery)
            print("[TEST]")
            print(test_result)

            return device_id, battery, test_result

        except Exception as e:
            print(f"Переподключение не удалось: {e}")
            time.sleep(RECONNECT_DELAY_SEC)


def try_get_measurements_after_reconnect(
    device: P334,
) -> tuple[float | None, float | None, float | None, float | None, str]:
    """
    После переподключения пробуем всё равно получить AVR и MAX.
    Даже если ответы "съедут", collect_measurements разложит их по местам.
    """
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
    print(f"Тайм-аут ответа команды: {str(COMMAND_RESPONSE_TIMEOUT_SEC).replace('.', ',')} сек")

    device = P334(PORT, baudrate=BAUDRATE, timeout=TIMEOUT)

    try:
        device_id, battery0, test_result = connect_and_initialize(device)
        set_frequency_if_needed(device)
        device.reset_buffers()

        print("\nПодключение выполнено.")
        print("\n[ID]")
        print(device_id)
        print("\n[UBATT]")
        print(battery0)
        print("\n[TEST]")
        print(test_result)

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
                    f"Сохранено: {interval_started} | {interval_finished} | "
                    f"AVG={average_value}±{average_error} | "
                    f"MAX={max_value}±{max_error} {unit}"
                )

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

                print("Строка с ERROR записана в файл.")
                cycle_number += 1

    except KeyboardInterrupt:
        print("\nСкрипт остановлен пользователем.")
        print(f"Последние данные сохранены в: {output_path}")

    finally:
        device.close()


if __name__ == "__main__":
    main()
