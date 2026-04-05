# MTPanel

Self-hosted веб-панель для управления MTProxy (`Go + SvelteKit + SQLite + systemd`).

## Установка одной строкой

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash
```

С указанием портов:

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash -s -- --port 8080 --mtproxy-port 443
```

## Быстрая локальная установка (из этого репозитория)

```bash
sudo bash ./install.sh --port 8080 --mtproxy-port 443
```

## Что делает установщик

1. Определяет ОС/архитектуру и проверяет наличие `systemd`.
2. Создаёт системного пользователя и нужные директории.
3. Пытается скачать бинарник панели из GitHub Release.
4. Если релиз/бинарник отсутствует, автоматически собирает `mtpanel` из исходников репозитория.
5. Пишет конфиг в `/etc/mtpanel/config.json`.
6. Устанавливает и запускает `mtpanel.service`.
7. Показывает URL панели и начальный пароль.

## Smoke-тест (основной сценарий)

1. Откройте `http://<SERVER_IP>:8080`.
2. Перейдите на `/setup` и задайте пароль администратора.
3. Выполните вход.
4. Откройте страницу `Proxy` и нажмите `Install MTProxy`.
5. Проверьте, что статус стал `running`.
6. Откройте `Links`, создайте ссылку, скопируйте/поделитесь.
7. Отзовите одну ссылку и проверьте, что она стала неактивной.
8. Откройте `Logs` и убедитесь, что видны логи MTProxy.
9. Откройте `Updates` и запустите проверку обновлений.
10. Откройте `Settings`, измените порт MTProxy и пароль.

## Полезные команды сервисов

```bash
sudo systemctl status mtpanel
sudo systemctl restart mtpanel
sudo journalctl -u mtpanel -n 200 --no-pager

sudo systemctl status mtproto-proxy.service
sudo journalctl -u mtproto-proxy.service -n 200 --no-pager
```

## Удаление (вручную)

```bash
sudo systemctl stop mtpanel
sudo systemctl disable mtpanel
sudo rm -f /etc/systemd/system/mtpanel.service
sudo systemctl daemon-reload
```

## Примечания

- Поддерживается только Linux + systemd.
- Для production рекомендуется reverse proxy + TLS.
- Установщик идемпотентный, его можно запускать повторно для обновлений.
