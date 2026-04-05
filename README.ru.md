# MTPanel (RU)

Self-hosted веб-панель для управления **TeleMT** на Linux-сервере.

## Содержание

1. [Что это](#что-это)
2. [Возможности](#возможности)
3. [Требования](#требования)
4. [Установка в одну строку](#установка-в-одну-строку)
5. [Параметры установщика](#параметры-установщика)
6. [Что делается автоматически](#что-делается-автоматически)
7. [Первый запуск](#первый-запуск)
8. [Скриншоты](#скриншоты)
9. [Полное удаление](#полное-удаление)
10. [Диагностика](#диагностика)

## Что это

MTPanel дает простой сценарий:

- установить одной командой;
- открыть UI;
- управлять TeleMT;
- генерировать/отзывать proxy-ссылки.

## Возможности

- Первичная настройка и авторизация.
- Управление TeleMT (`install/start/stop/restart`).
- Генерация `tg://proxy?...` ссылок в 1 клик.
- Просмотр статуса и логов.
- Проверка/применение обновлений TeleMT.
- Автоматическая настройка firewall в installer.

## Требования

- Linux + `systemd`.
- Root (`sudo`).

## Установка в одну строку

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash
```

С явными параметрами:

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash -s -- --port 8080 --proxy-port 443
```

С ограничением доступа к панели:

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash -s -- --panel-allow 1.2.3.4/32
```

## Параметры установщика

- `--port <port>` — порт панели (по умолчанию `8080`).
- `--proxy-port <port>` — порт TeleMT (по умолчанию `443`).
- `--panel-allow <CIDR>` — сеть, которой разрешен доступ к панели.
- `--repo <owner/repo>` — репозиторий релизов.

## Что делается автоматически

Installer:

1. Проверяет ОС/архитектуру/зависимости.
2. Скачивает или собирает backend и frontend.
3. Пишет конфиг в `/etc/mtpanel/config.json`.
4. Удаляет старые `mtpanel.service.d` overrides.
5. Ставит стабильный `mtpanel.service`.
6. Применяет firewall:
   - `proxy-port` открыт для всех;
   - `panel-port` ограничен SSH IP (или открыт всем, если IP не определился).

## Первый запуск

1. Откройте `http://<SERVER_IP>:8080`.
2. Перейдите на `/setup` и задайте пароль администратора.
3. Войдите через `/login`.
4. В разделе `Прокси` установите TeleMT.

## Скриншоты

Разместите файлы в `docs/screenshots/`:

```text
docs/screenshots/dashboard.png
docs/screenshots/proxy.png
docs/screenshots/links.png
docs/screenshots/updates.png
```

Галерея:

![Обзор](docs/screenshots/dashboard.png)
![Прокси](docs/screenshots/proxy.png)
![Ссылки](docs/screenshots/links.png)
![Обновления](docs/screenshots/updates.png)

## Полное удаление

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

## Диагностика

```bash
sudo systemctl status mtpanel --no-pager -l
sudo journalctl -u mtpanel -n 120 --no-pager -l
curl -I http://127.0.0.1:8080/
```

---

- Репозиторий: <https://github.com/NikitaKHS/mtpanel>
- English: [README.en.md](README.en.md)
