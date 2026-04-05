# MTPanel

Self-hosted веб-панель для управления **TeleMT** (`Go + SvelteKit + SQLite + systemd`).

## Установка в одну строку

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash
```

С явной настройкой портов:

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash -s -- --port 8080 --proxy-port 443
```

## Что делает установщик

1. Проверяет ОС/архитектуру и наличие `systemd`.
2. Создает системные директории и пользователя `mtpanel`.
3. Скачивает бинарник панели (или собирает из исходников, если релиз недоступен).
4. Ставит фронтенд-ассеты.
5. Пишет конфиг панели в `/etc/mtpanel/config.json`.
6. Удаляет старые `mtpanel.service.d/override.conf` (если есть).
7. Ставит и запускает минимальный стабильный `mtpanel.service` (без проблемного hardening).

## Первый запуск

1. Откройте `http://<SERVER_IP>:8080`.
2. При первом входе откройте `/setup` и задайте пароль администратора.
3. После этого вход через `/login`.
4. В разделе `Прокси` установите TeleMT.

## Быстрая проверка

```bash
sudo systemctl status mtpanel --no-pager -l
sudo journalctl -u mtpanel -n 120 --no-pager -l
curl -I http://127.0.0.1:8080/
```

## Полное удаление (для чистого теста)

Внимание: команда удалит панель, базу и настройки.

```bash
sudo systemctl stop mtpanel 2>/dev/null || true
sudo systemctl disable mtpanel 2>/dev/null || true
sudo pkill -f '/opt/mtpanel/mtpanel' || true

sudo systemctl stop telemt 2>/dev/null || true
sudo systemctl disable telemt 2>/dev/null || true

sudo rm -f /etc/systemd/system/mtpanel.service
sudo rm -rf /etc/systemd/system/mtpanel.service.d
sudo rm -f /etc/systemd/system/telemt.service
sudo systemctl daemon-reload

sudo rm -rf /opt/mtpanel /opt/telemt
sudo rm -rf /etc/mtpanel /etc/telemt
sudo rm -rf /var/lib/mtpanel

sudo userdel mtpanel 2>/dev/null || true
```

После этого можно ставить заново:

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash
```

## Примечания

- Поддерживается Linux + systemd.
- Используется только `telemt.service` (режим `telemt-only`).
- Для production рекомендуется reverse proxy + TLS.
